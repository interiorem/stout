package porto

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/apex/log"
	apexctx "github.com/m0sth8/context"
	"github.com/mitchellh/mapstructure"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/docker"
	"github.com/noxiouz/stout/pkg/semaphore"

	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"

	_ "github.com/docker/distribution/manifest/schema1"
	_ "github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/transport"
	engineref "github.com/docker/engine-api/types/reference"
)

type portoBoxConfig struct {
	// Directory where volumes per app are placed
	Layers string `json:"layers"`
	// Directory for containers
	Containers string `json:"containers"`

	SpawnConcurrency uint              `json:"concurrency"`
	RegistryAuth     map[string]string `json:"registryauth"`
}

func (cfg *portoBoxConfig) ContainerRootDir(name, containerID string) string {
	return filepath.Join(cfg.Containers, name, containerID)
}

// Box operates with Porto to launch containers
type Box struct {
	config     *portoBoxConfig
	instanceID string

	spawnSM      semaphore.Semaphore
	transport    *http.Transport
	muContainers sync.Mutex
	containers   map[string]*container
	blobRepo     BlobRepository
}

// NewBox creates new Box
func NewBox(ctx context.Context, cfg isolate.BoxConfig) (isolate.Box, error) {
	var config = &portoBoxConfig{
		SpawnConcurrency: 10,
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

	apexctx.GetLogger(ctx).WithField("dir", config.Layers).Info("create directory for Layers")
	if err = os.MkdirAll(config.Layers, 0755); err != nil {
		return nil, err
	}

	apexctx.GetLogger(ctx).WithField("dir", config.Containers).Info("create directory for Containers")
	if err = os.MkdirAll(config.Containers, 0755); err != nil {
		return nil, err
	}

	blobRepo, err := NewBlobRepository(ctx, BlobRepositoryConfig{SpoolPath: config.Layers})
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			dialer := net.Dialer{
				DualStack: true,
				Timeout:   5 * time.Second,
			}
			return dialer.Dial(network, addr)
		},
	}

	box := &Box{
		config:     config,
		instanceID: uuid.New(),
		transport:  tr,
		spawnSM:    semaphore.New(config.SpawnConcurrency),
		containers: make(map[string]*container),

		blobRepo: blobRepo,
	}

	return box, nil
}

func (b *Box) appLayerName(appname string) string {
	return b.instanceID + appname
}

// Spool downloades Docker images from Distribution, builds base layer for Porto container
func (b *Box) Spool(ctx context.Context, name string, opts isolate.Profile) (err error) {
	profile, err := docker.ConvertProfile(opts)
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithField("name", name).Info("unbale to convert raw profile to Porto/Docker specific profile")
		return err
	}

	if profile.Registry == "" {
		apexctx.GetLogger(ctx).WithField("name", name).Error("Registry must be non empty")
		return fmt.Errorf("Registry must be non empty")
	}

	portoConn, err := porto.Connect()
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithField("name", name).Error("Porto connection error")
		return err
	}

	named, err := reference.ParseNamed(filepath.Join(profile.Repository, profile.Repository, name))
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithField("name", name).Error("name is invalid")
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

	layerName := b.appLayerName(name)
	if err = portoConn.RemoveLayer(layerName); err != nil && !isEqualPortoError(err, portorpc.EError_LayerNotFound) {
		return err
	}

	for _, descriptor := range manifest.References() {
		blobPath, err := b.blobRepo.Get(ctx, repo, descriptor.Digest)
		if err != nil {
			return err
		}
		if err = portoConn.ImportLayer(layerName, blobPath, true); err != nil {
			return err
		}
	}

	return nil
}

// Spawn spawns new Porto container
func (b *Box) Spawn(ctx context.Context, config isolate.SpawnConfig, output io.Writer) (isolate.Process, error) {
	profile, err := docker.ConvertProfile(config.Opts)
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithFields(log.Fields{"name": config.Name}).Info("unable to convert raw profile to Docker specific profile")
		return nil, err
	}
	start := time.Now()

	spawningQueueSize.Inc(1)
	if spawningQueueSize.Count() > 10 {
		spawningQueueSize.Dec(1)
		return nil, syscall.EAGAIN
	}

	ei := execInfo{
		Profile:    profile,
		name:       config.Name,
		executable: config.Executable,
		args:       config.Args,
		env:        config.Env,
	}

	ID := uuid.New()
	cfg := containerConfig{
		Root:  filepath.Join(b.config.Containers, ID),
		ID:    ID,
		Layer: b.appLayerName(config.Name),
	}

	portoConn, err := porto.Connect()
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

	containersCreatedCounter.Inc(1)
	pr, err := newContainer(ctx, portoConn, cfg, ei)
	if err != nil {
		containersErroredCounter.Inc(1)
		return nil, err
	}

	b.muContainers.Lock()
	b.containers[pr.containerID] = pr
	b.muContainers.Unlock()

	if err = pr.start(portoConn, output); err != nil {
		containersErroredCounter.Inc(1)
		return nil, err
	}
	isolate.NotifyAbouStart(output)
	totalSpawnTimer.UpdateSince(start)
	return pr, nil
}

// Close releases all resources such as idle connections from http.Transport
func (b *Box) Close() error {
	b.transport.CloseIdleConnections()
	return nil
}
