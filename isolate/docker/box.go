package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/apex/log"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
)

const (
	dockerAPIVersion = "v1.19"
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
	client *client.Client
}

type dockerBoxConfig struct {
	DockerEndpoint string `json:"endpoint"`
	APIVersion     string `json:"version"`
}

// NewBox ...
func NewBox(cfg isolate.BoxConfig) (isolate.Box, error) {
	var config = &dockerBoxConfig{
		DockerEndpoint: client.DefaultDockerHost,
		APIVersion:     dockerAPIVersion,
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

	return &Box{client: client}, nil
}

// Spawn spawns a prcess using container
func (b *Box) Spawn(ctx context.Context, opts isolate.Profile, name, executable string, args, env map[string]string) (isolate.Process, error) {
	profile, err := convertProfile(opts)
	if err != nil {
		isolate.GetLogger(ctx).WithError(err).WithFields(log.Fields{"name": name}).Info("unable to convert raw profile to Docker specific profile")
		return nil, err
	}
	return newContainer(ctx, b.client, profile, name, executable, args, env)
}

// Spool spools an image with a tag latest
func (b *Box) Spool(ctx context.Context, name string, opts isolate.Profile) (err error) {
	profile, err := convertProfile(opts)
	if err != nil {
		isolate.GetLogger(ctx).WithError(err).WithFields(log.Fields{"name": name}).Info("unbale to convert raw profile to Docker specific profile")
		return err
	}

	if profile.Registry == "" {
		isolate.GetLogger(ctx).WithFields(log.Fields{"name": name}).Info("local image will be used")
		return nil
	}

	defer isolate.GetLogger(ctx).WithField("name", name).Trace("spooling an image").Stop(&err)

	pullOpts := types.ImagePullOptions{
		ImageID: filepath.Join(profile.Registry, profile.Repository, name),
		Tag:     "latest",
	}

	body, err := b.client.ImagePull(ctx, pullOpts, nil)
	if err != nil {
		isolate.GetLogger(ctx).WithError(err).WithFields(
			log.Fields{"name": name, "image": pullOpts.ImageID, "tag": pullOpts.Tag}).Error("unable to pull an image")
		return err
	}
	defer body.Close()

	var (
		resp   spoolResponseProtocol
		logger = isolate.GetLogger(ctx).WithField("name", name)
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
