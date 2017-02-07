package porto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/log"
	"github.com/noxiouz/stout/pkg/semaphore"
	"github.com/pborman/uuid"
	"github.com/uber-go/zap"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/transport"
	engineref "github.com/docker/engine-api/types/reference"
	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

type portoBoxConfig struct {
	// Directory where volumes per app are placed
	Layers string `json:"layers"`
	// Directory for containers
	Containers string `json:"containers"`
	// Path to a journal file
	Journal string `json:"journal"`

	SpawnConcurrency uint              `json:"concurrency"`
	RegistryAuth     map[string]string `json:"registryauth"`
	DialRetries      int               `json:"dialretries"`
	CleanupEnabled   bool              `json:"cleanupenabled"`
	SetImgURI        bool              `json:"setimguri"`
	WeakEnabled      bool              `json:"weakenabled"`
	DefaultUlimits   string            `json:"defaultulimits"`
	VolumeBackend    string            `json:"volumebackend"`
}

func (c *portoBoxConfig) String() string {
	body, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(body)
}

func (c *portoBoxConfig) ContainerRootDir(name, containerID string) string {
	return filepath.Join(c.Containers, name, containerID)
}

// Box operates with Porto to launch containers
type Box struct {
	config  *portoBoxConfig
	journal *journal

	spawnSM      semaphore.Semaphore
	transport    *http.Transport
	muContainers sync.Mutex
	containers   map[string]*container
	blobRepo     BlobRepository

	rootPrefix string

	onClose context.CancelFunc
}

const defaultVolumeBackend = "overlay"

// NewBox creates new Box
func NewBox(ctx context.Context, cfg isolate.BoxConfig) (isolate.Box, error) {
	var config = &portoBoxConfig{
		SpawnConcurrency: 10,
		DialRetries:      10,

		CleanupEnabled: true,
		WeakEnabled:    false,
	}
	decoderConfig := mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           config,
		TagName:          "json",
	}
	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return nil, err
	}

	if err = decoder.Decode(cfg); err != nil {
		return nil, err
	}

	if config.Layers == "" {
		return nil, fmt.Errorf("option Layers is invalid or unspecified")
	}
	if config.Containers == "" {
		return nil, fmt.Errorf("option Containers is invalid or unspecified")
	}

	if config.Journal == "" {
		return nil, fmt.Errorf("option Journal is empty or unspecified")
	}

	if config.VolumeBackend == "" {
		config.VolumeBackend = defaultVolumeBackend
	}

	log.G(ctx).Info("create directory for Layers", zap.String("dir", config.Layers))
	if err = os.MkdirAll(config.Layers, 0755); err != nil {
		return nil, err
	}

	log.G(ctx).Info("create directory for Containers", zap.String("dir", config.Containers))
	if err = os.MkdirAll(config.Containers, 0755); err != nil {
		return nil, err
	}

	blobRepo, err := NewBlobRepository(ctx, BlobRepositoryConfig{SpoolPath: config.Layers})
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			for i := 0; i <= config.DialRetries; i++ {
				dialer := net.Dialer{
					DualStack: true,
					Timeout:   5 * time.Second,
				}
				conn, err := dialer.Dial(network, addr)
				if err == nil {
					return conn, err
				}
				sleepTime := time.Duration(rand.Int63n(500)) * time.Millisecond
				log.G(ctx).Error("dial error, sleep", zap.Error(err), zap.String("net", network),
					zap.String("addr", addr), zap.Duration("sleep", sleepTime))
				time.Sleep(sleepTime)
			}
			return nil, fmt.Errorf("no retries available")
		},
	}

	portoConn, err := portoConnect()
	if err != nil {
		return nil, err
	}
	defer portoConn.Close()

	rootPrefix, err := portoConn.GetProperty("self", "absolute_name")
	if err != nil {
		return nil, err
	}
	if rootPrefix == "/" {
		rootPrefix = ""
	}

	ctx, onClose := context.WithCancel(ctx)
	box := &Box{
		config:     config,
		journal:    newJournal(),
		transport:  tr,
		spawnSM:    semaphore.New(config.SpawnConcurrency),
		containers: make(map[string]*container),
		onClose:    onClose,
		rootPrefix: rootPrefix,

		blobRepo: blobRepo,
	}

	body, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	portoConfig.Set(string(body))

	if err = box.loadJournal(ctx); err != nil {
		box.Close()
		return nil, err
	}

	layers, err := portoConn.ListLayers()
	if err != nil {
		return nil, err
	}

	box.journal.UpdateFromPorto(layers)

	journalContent.Set(box.journal.String())

	go box.waitLoop(ctx)
	go box.dumpJournalEvery(ctx, time.Minute)

	return box, nil
}

func (b *Box) dumpJournalEvery(ctx context.Context, every time.Duration) {
	for {
		select {
		case <-time.After(every):
			b.dumpJournal(ctx)
		case <-ctx.Done():
			b.dumpJournal(ctx)
			return
		}
	}
}

func (b *Box) dumpJournal(ctx context.Context) (err error) {
	log.G(ctx).Info("dumping journal")
	tempfile, err := ioutil.TempFile(filepath.Dir(b.config.Journal), "portojournalbak")
	if err != nil {
		log.G(ctx).Error("creating temp journal file failed", zap.Error(err))
		return err
	}
	defer os.Remove(tempfile.Name())
	defer tempfile.Close()

	if err = b.journal.Dump(tempfile); err != nil {
		log.G(ctx).Error("dump failed", zap.Error(err))
		return err
	}

	if err = os.Rename(tempfile.Name(), b.config.Journal); err != nil {
		log.G(ctx).Error("moving temp journal failed", zap.Error(err))
		return err
	}

	return nil
}

func (b *Box) loadJournal(ctx context.Context) error {
	f, err := os.Open(b.config.Journal)
	if err != nil {
		log.G(ctx).Warn("unable to open Journal file", zap.Error(err))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	if err = b.journal.Load(f); err != nil {
		log.G(ctx).Error("unable to load Journal", zap.Error(err))
		return err
	}

	return nil
}

func (b *Box) waitLoop(ctx context.Context) {
	log.G(ctx).Info("start waitLoop")
	var (
		portoConn porto.API
		err       error
	)

	waitPattern := filepath.Join(b.rootPrefix, "*")

	var waitTimeout = 30 * time.Second

	closed := func(portoConn porto.API) bool {
		select {
		case <-ctx.Done():
			if portoConn != nil {
				portoConn.Close()
			}
			return true
		default:
			return false
		}
	}

LOOP:
	for {
		log.G(ctx).Info("next iteration of waitLoop")
		if closed(portoConn) {
			return
		}
		// Connect to Porto if we have not connected yet.
		// In case of error: wait either a fixed timeout or closing of Box
		if portoConn == nil {
			log.G(ctx).Info("waitLoop: connect to Portod")
			portoConn, err = portoConnect()
			if err != nil {
				log.G(ctx).Warn("unable to connect to Portod", zap.Error(err))
				select {
				case <-time.After(time.Second):
					continue LOOP
				case <-ctx.Done():
					return
				}
			}
		}

		// * means all containers
		// if no containers dead for waitTimeout, name will be an empty string
		containerName, err := portoConn.Wait([]string{waitPattern}, 30*waitTimeout)
		if err != nil {
			portoConn.Close()
			portoConn = nil
			continue LOOP
		}

		if containerName != "" {
			log.G(ctx).Info("Wait reports that container is dead", zap.String("container", containerName))
			b.muContainers.Lock()
			container, ok := b.containers[containerName]
			if ok {
				delete(b.containers, containerName)
			}
			rest := len(b.containers)
			b.muContainers.Unlock()
			if ok {
				if err = container.Kill(); err != nil {
					log.G(ctx).Error("Killing error", zap.String("container", containerName), zap.Error(err))
				}
			}

			log.G(ctx).Info("containers are being tracked now", zap.Int("amount", rest))
		}
	}
}

func (b *Box) appGenLabel(appname string) string {
	appname = strings.Replace(appname, ":", "_", -1)
	return appname
}

// func (b *Box) appLayerName(appname string) string {
// 	if b.config.WeakEnabled {
// 		return "_weak_" + b.appGenLabel(appname) + "_" + b.journal.UUID
// 	}
// 	return b.appGenLabel(appname) + "_" + b.journal.UUID
// }

func (b *Box) addRootNamespacePrefix(container string) string {
	return filepath.Join(b.rootPrefix, container)
}

// Spool downloades Docker images from Distribution, builds base layer for Porto container
func (b *Box) Spool(ctx context.Context, name string, opts isolate.RawProfile) (err error) {
	lg := log.G(ctx).With(zap.String("name", name))
	lg.Info("start spooling")

	var profile = new(Profile)
	if err = opts.DecodeTo(profile); err != nil {
		lg.Info("unbale to convert raw profile to Porto/Docker specific profile", zap.Error(err))
		return err
	}

	if profile.Registry == "" {
		lg.Error("Registry must be non empty")
		return fmt.Errorf("Registry must be non empty")
	}

	named, err := reference.ParseNamed(filepath.Join(profile.Repository, profile.Repository, name))
	if err != nil {
		lg.Error("name is invalid", zap.Error(err))
		return err
	}

	portoConn, err := portoConnect()
	if err != nil {
		lg.Error("Porto connection error", zap.Error(err))
		return err
	}

	var tr http.RoundTripper
	if registryAuth, ok := b.config.RegistryAuth[profile.Registry]; ok {
		tr = transport.NewTransport(b.transport, transport.NewHeaderRequestModifier(http.Header{
			"Authorization": []string{registryAuth},
		}))
	} else {
		tr = b.transport
	}

	var registry = profile.Registry
	if !strings.HasPrefix(registry, "http") {
		registry = "https://" + registry
	}
	lg.Debug("Image URI generated", zap.String("registry", registry), zap.Stringer("name", named))
	repo, err := client.NewRepository(ctx, named, registry, tr)
	if err != nil {
		return err
	}

	tagDescriptor, err := repo.Tags(ctx).Get(ctx, engineref.GetTagFromNamedRef(named))
	if err != nil {
		return err
	}

	manifests, err := repo.Manifests(ctx)
	if err != nil {
		return err
	}

	manifest, err := manifests.Get(ctx, tagDescriptor.Digest)
	if err != nil {
		return err
	}

	var order layersOrder
	switch manifest.(type) {
	case schema1.SignedManifest, *schema1.SignedManifest:
		order = layerOrderV1
	case schema2.DeserializedManifest, *schema2.DeserializedManifest:
		order = layerOrderV2
	default:
		return fmt.Errorf("unknown manifest type %T", manifest)
	}

	layers := make([]string, 0)

	for _, descriptor := range order(manifest.References()) {
		// TODO: Add support for __weak__ layers
		layerName := descriptor.Digest.String()
		// TODO: insert check of the layer existance here
		// ListLayers is too heavy IMHO
		// if the layer presents we can skip it
		blobPath, err := b.blobRepo.Get(ctx, repo, descriptor.Digest)
		if err != nil {
			return err
		}

		lg.Info("import layer to Porto", zap.String("layer", layerName))
		portoLayerName := strings.Replace(layerName, ":", "_", -1)
		err = portoConn.ImportLayer(portoLayerName, blobPath, false)
		if err != nil && !isEqualPortoError(err, portorpc.EError_LayerAlreadyExists) {
			lg.Error("import layer failed", zap.String("layer", layerName), zap.Error(err))
			return err
		}
		lg.Info("layer successfully imported", zap.String("layer", layerName))
		layers = append(layers, portoLayerName)
	}
	b.journal.InsertManifestLayers(name, strings.Join(layers, ";"))
	// NOTE: Not so fast, but it's important for debug
	journalContent.Set(b.journal.String())
	return nil
}

// Spawn spawns new Porto container
func (b *Box) Spawn(ctx context.Context, config isolate.SpawnConfig, output io.Writer) (isolate.Process, error) {
	var profile = new(Profile)
	err := config.Opts.DecodeTo(profile)
	if err != nil {
		log.G(ctx).Error("unable to decode profile", zap.Error(err))
		return nil, err
	}
	start := time.Now()

	spawningQueueSize.Inc(1)
	if spawningQueueSize.Count() > 10 {
		spawningQueueSize.Dec(1)
		return nil, syscall.EAGAIN
	}

	layers := b.journal.GetManifestLayers(config.Name)
	if layers == "" {
		err = fmt.Errorf("no layers in the journal for the app")
		log.G(ctx).Error("unable to start container", zap.String("name", config.Name), zap.Error(err))
		return nil, err
	}

	ID := b.appGenLabel(config.Name) + "_" + uuid.New()
	cfg := containerConfig{
		Root:           filepath.Join(b.config.Containers, ID),
		ID:             b.addRootNamespacePrefix(ID),
		Layer:          layers,
		CleanupEnabled: b.config.CleanupEnabled,
		SetImgURI:      b.config.SetImgURI,
		VolumeBackend:  b.config.VolumeBackend,
		execInfo: execInfo{
			Profile:    profile,
			name:       config.Name,
			executable: config.Executable,
			ulimits:    b.config.DefaultUlimits,
			args:       config.Args,
			env:        config.Env,
		},
	}

	portoConn, err := portoConnect()
	if err != nil {
		return nil, err
	}
	defer portoConn.Close()

	err = b.spawnSM.Acquire(ctx)
	spawningQueueSize.Dec(1)
	if err != nil {
		return nil, isolate.ErrSpawningCancelled
	}
	defer b.spawnSM.Release()

	if cm := log.G(ctx).Check(zap.InfoLevel, "create container"); cm.OK() {
		cm.Write(zap.String("id", cfg.ID), zap.String("name", config.Name), zap.String("root", cfg.Root))
	}

	containersCreatedCounter.Inc(1)
	pr, err := newContainer(ctx, portoConn, cfg)
	if err != nil {
		containersErroredCounter.Inc(1)
		return nil, err
	}

	b.muContainers.Lock()
	b.containers[pr.containerID] = pr
	b.muContainers.Unlock()

	if err = pr.start(portoConn, output); err != nil {
		containersErroredCounter.Inc(1)
		pr.Cleanup(portoConn)
		return nil, err
	}
	isolate.NotifyAboutStart(output)
	totalSpawnTimer.UpdateSince(start)
	return pr, nil
}

// Close releases all resources such as idle connections from http.Transport
func (b *Box) Close() error {
	b.transport.CloseIdleConnections()
	b.onClose()
	return nil
}
