package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/noxiouz/stout/pkg/log"
	"golang.org/x/net/context"

	apexlog "github.com/apex/log"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/mitchellh/mapstructure"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/semaphore"
)

const (
	defaultSpawnConcurrency = 10

	isolateDockerLabel = "cocaine-isolate"
)

var (
	defaultHeaders = map[string]string{"User-Agent": "cocaine-universal-isolate"}
)

type spoolResponseProtocol struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

// Box ...
type Box struct {
	ctx          context.Context
	cancellation context.CancelFunc

	client *client.Client

	spawnSM semaphore.Semaphore

	config *dockerBoxConfig

	state   isolate.GlobalState

	muContainers sync.Mutex
	containers   map[string]*process
}

type dockerBoxConfig struct {
	DockerEndpoint   string            `json:"endpoint"`
	APIVersion       string            `json:"version"`
	SpawnConcurrency uint              `json:"concurrency"`
	RegistryAuth     map[string]string `json:"registryauth"`
}

// NewBox ...
func NewBox(ctx context.Context, cfg isolate.BoxConfig, gstate isolate.GlobalState) (isolate.Box, error) {
	var config = &dockerBoxConfig{
		DockerEndpoint:   client.DefaultDockerHost,
		SpawnConcurrency: defaultSpawnConcurrency,
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

	client, err := client.NewClient(config.DockerEndpoint, config.APIVersion, nil, defaultHeaders)
	if err != nil {
		return nil, err
	}

	ctx, cancellation := context.WithCancel(ctx)
	box := &Box{
		ctx:          ctx,
		cancellation: cancellation,

		client:     client,
		spawnSM:    semaphore.New(config.SpawnConcurrency),
		config:     config,
		state: gstate,
		containers: make(map[string]*process),
	}

	body, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	dockerConfig.Set(string(body))

	go box.watchEvents()

	return box, nil
}

func (b *Box) watchEvents() {
	const dieEvent = "die"

	since := time.Now()
	sleep := time.Second
	maxSleep := time.Second * 32

	filterArgs := filters.NewArgs()
	filterArgs.Add("event", dieEvent)
	filterArgs.Add("label", isolateDockerLabel)
	fltrs, _ := filters.ToParam(filterArgs)

	var eventResponse struct {
		Status string `json:"status"`
		ID     string `json:"id"`
		Time   int64  `json:"time"`
	}

	logger := log.G(b.ctx)

	for {
		eventsOptions := types.EventsOptions{
			Since:   strconv.FormatInt(since.Unix(), 10),
			Filters: filterArgs,
		}

		logger.Infof("listening Docker events since %s with filters %s", eventsOptions.Since, fltrs)
		resp, err := b.client.Events(b.ctx, eventsOptions)
		switch err {
		case nil:
			sleep = time.Second
			decoder := json.NewDecoder(resp)
			for {
				if err = decoder.Decode(&eventResponse); err != nil {
					logger.WithError(err).Error("unable to decode Docker events")
					resp.Close()
					break
				}

				// Save timestamp of the latest received event
				since = time.Unix(eventResponse.Time, 0)

				switch eventResponse.Status {
				case dieEvent:
					logger.WithField("id", eventResponse.ID).Info("container has died")

					var p *process
					b.muContainers.Lock()
					p, ok := b.containers[eventResponse.ID]
					delete(b.containers, eventResponse.ID)
					b.muContainers.Unlock()
					if ok {
						p.remove()
					} else {
						// NOTE: it could be orphaned worker from our previous launch
						logger.WithField("id", eventResponse.ID).Warn("unknown container will be removed")
						containerRemove(b.client, b.ctx, eventResponse.ID)
					}

				default:
					logger.WithField("status", eventResponse.Status).Warn("unknown status")
				}
			}

		case context.Canceled, context.DeadlineExceeded:
			logger.Info("event listenening has been cancelled")
			return

		default:
			// backoff
			sleep *= 2
			if sleep > maxSleep {
				sleep = maxSleep
			}
			logger.WithError(err).Warnf("unable to listen events. Sleep %s", sleep)
			time.Sleep(sleep)
		}
	}
}

// Close releases all resources connected to the Box
func (b *Box) Close() error {
	b.cancellation()
	return nil
}

// Spawn spawns a prcess using container
func (b *Box) Spawn(ctx context.Context, config isolate.SpawnConfig, output io.Writer) (isolate.Process, error) {
	profile, err := decodeProfile(config.Opts)
	if err != nil {
		log.G(ctx).WithError(err).WithFields(apexlog.Fields{"name": config.Name}).Info("unable to convert raw profile to Docker specific profile")
		return nil, err
	}
	start := time.Now()

	spawningQueueSize.Inc(1)
	if spawningQueueSize.Count() > 10 {
		spawningQueueSize.Dec(1)
		return nil, syscall.EAGAIN
	}
	err = b.spawnSM.Acquire(ctx)
	spawningQueueSize.Dec(1)
	if err != nil {
		return nil, isolate.ErrSpawningCancelled
	}
	defer b.spawnSM.Release()

	containersCreatedCounter.Inc(1)
	pr, err := newContainer(ctx, b.client, profile, config.Name, config.Executable, config.Args, config.Env)
	if err != nil {
		containersErroredCounter.Inc(1)
		return nil, err
	}

	b.muContainers.Lock()
	b.containers[pr.containerID] = pr
	b.muContainers.Unlock()

	if err = pr.startContainer(output); err != nil {
		containersErroredCounter.Inc(1)
		return nil, err
	}

	totalSpawnTimer.UpdateSince(start)
	return pr, nil
}

func (b *Box) Inspect(ctx context.Context, workeruuid string) ([]byte, error) {
	b.muContainers.Lock()
	for cid, container := range b.containers {
		if container.uuid == workeruuid {
			b.muContainers.Unlock()
			_, data, err := b.client.ContainerInspectWithRaw(ctx, cid, false)
			return data, err
		}
	}
	b.muContainers.Unlock()
	return []byte("{}"), nil
}

// Spool spools an image with a tag latest
func (b *Box) Spool(ctx context.Context, name string, opts isolate.RawProfile) (err error) {
	profile, err := decodeProfile(opts)
	if err != nil {
		log.G(ctx).WithError(err).WithFields(apexlog.Fields{"name": name}).Info("unbale to convert raw profile to Docker specific profile")
		return err
	}

	if profile.Registry == "" {
		log.G(ctx).WithField("name", name).Info("local image will be used")
		return nil
	}

	ref := filepath.Join(profile.Registry, profile.Repository, name)

	defer log.G(ctx).WithField("ref", ref).Trace("spooling an image").Stop(&err)
	pullOpts := types.ImagePullOptions{
		All: false,
	}

	if registryAuth, ok := b.config.RegistryAuth[profile.Registry]; ok {
		pullOpts.RegistryAuth = registryAuth
	}

	body, err := b.client.ImagePull(ctx, ref, pullOpts)
	if err != nil {
		log.G(ctx).WithError(err).WithField("ref", ref).Error("unable to pull an image")
		return err
	}
	defer body.Close()

	if err = decodeImagePull(ctx, body); err != nil {
		return err
	}

	return nil
}

// decodeImagePull detects Error of an image pulling proces
// by decoding reply from Docker
// Although Docker should reply with JSON Encoded items
// one per line, in different versions it could vary.
// This decoders can detect error even in mixed replies:
// {"Status": "OK"}\n{"Status": "OK"}
// {"Status": "OK"}{"Error": "error"}
func decodeImagePull(ctx context.Context, r io.Reader) error {
	logger := log.G(ctx)
	more := true

	rd := bufio.NewReader(r)
	for more {
		line, err := rd.ReadBytes('\n')
		switch err {
		case nil:
			// pass
		case io.EOF:
			if len(line) == 0 {
				return nil
			}
			more = false
		default:
			return err
		}

		if len(line) == 0 {
			return fmt.Errorf("Empty response line")
		}

		if line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		if err = decodePullLine(line); err != nil {
			logger.WithError(err).Errorf("unable to decode JSON docker reply")
			return err
		}
	}
	return nil
}

func decodePullLine(line []byte) error {
	var resp spoolResponseProtocol
	decoder := json.NewDecoder(bytes.NewReader(line))
	for {
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		if len(resp.Error) != 0 {
			return fmt.Errorf(resp.Error)
		}
	}
}
