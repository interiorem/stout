package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	apexctx "github.com/m0sth8/context"
	"golang.org/x/net/context"

	"github.com/apex/log"
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
func NewBox(ctx context.Context, cfg isolate.BoxConfig) (isolate.Box, error) {
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
		containers: make(map[string]*process),
	}

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

	logger := apexctx.GetLogger(b.ctx)

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
			logger.WithError(err).Errorf("unable to listen events. Sleep %s", sleep)
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
func (b *Box) Spawn(ctx context.Context, opts isolate.Profile, name, executable string, args, env map[string]string) (isolate.Process, error) {
	profile, err := convertProfile(opts)
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithFields(log.Fields{"name": name}).Info("unable to convert raw profile to Docker specific profile")
		return nil, err
	}
	start := time.Now()

	spawningQueueSize.Inc(1)
	err = b.spawnSM.Acquire(ctx)
	spawningQueueSize.Dec(1)
	if err != nil {
		return nil, err
	}
	defer b.spawnSM.Release()

	containersCreatedCounter.Inc(1)
	pr, err := newContainer(ctx, b.client, profile, name, executable, args, env)
	if err != nil {
		containersErroredCounter.Inc(1)
		return nil, err
	}

	b.muContainers.Lock()
	b.containers[pr.containerID] = pr
	b.muContainers.Unlock()

	if err = pr.startContainer(); err != nil {
		containersErroredCounter.Inc(1)
		return nil, err
	}

	totalSpawnTimer.UpdateSince(start)
	return pr, nil
}

// Spool spools an image with a tag latest
func (b *Box) Spool(ctx context.Context, name string, opts isolate.Profile) (err error) {
	profile, err := convertProfile(opts)
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithFields(log.Fields{"name": name}).Info("unbale to convert raw profile to Docker specific profile")
		return err
	}

	if profile.Registry == "" {
		apexctx.GetLogger(ctx).WithFields(log.Fields{"name": name}).Info("local image will be used")
		return nil
	}

	defer apexctx.GetLogger(ctx).WithField("name", name).Trace("spooling an image").Stop(&err)

	pullOpts := types.ImagePullOptions{
		All: false,
	}

	if registryAuth, ok := b.config.RegistryAuth[profile.Registry]; ok {
		pullOpts.RegistryAuth = registryAuth
	}

	ref := fmt.Sprintf("%s:%s", filepath.Join(profile.Registry, profile.Repository, name), "latest")

	body, err := b.client.ImagePull(ctx, ref, pullOpts)
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithFields(
			log.Fields{"name": name, "ref": ref}).Error("unable to pull an image")
		return err
	}
	defer body.Close()

	var (
		resp   spoolResponseProtocol
		logger = apexctx.GetLogger(ctx).WithField("name", name)
	)

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		if err = json.NewDecoder(bytes.NewReader(scanner.Bytes())).Decode(&resp); err != nil {
			logger.WithError(err).Errorf("unable to decode JSON docker reply %s", scanner.Bytes())
			return err
		}

		if len(resp.Error) != 0 {
			return fmt.Errorf("spooling error %s", resp.Error)
		}

		if len(resp.Status) != 0 {
			logger.Debugf("%s", resp.Status)
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return nil
}
