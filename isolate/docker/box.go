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
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
)

const (
	dockerVersionAPI = "v1.19"
)

var (
	defaultHeaders = map[string]string{"User-Agent": "cocaine-universal-isolate"}
)

type spoolResponseProtocol struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

// Box ...
type Box struct{}

// NewBox ...
func NewBox(cfg isolate.BoxConfig) (isolate.Box, error) {
	return &Box{}, nil
}

// Spawn spawns a prcess using container
func (b *Box) Spawn(ctx context.Context, opts isolate.Profile, name, executable string, args, env map[string]string) (isolate.Process, error) {
	profile, err := convertProfile(opts)
	if err != nil {
		isolate.GetLogger(ctx).WithError(err).WithFields(log.Fields{"name": name}).Info("unbale to convert raw profile to Docker specific profile")
		return nil, err
	}
	return newContainer(ctx, profile, name, executable, args, env)
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

	defer isolate.GetLogger(ctx).WithFields(log.Fields{"name": name, "endpoint": profile.Endpoint}).Trace("spooling an image").Stop(&err)

	cli, err := client.NewClient(profile.Endpoint, dockerVersionAPI, nil, defaultHeaders)
	if err != nil {
		return err
	}

	pullOpts := types.ImagePullOptions{
		ImageID: filepath.Join(profile.Registry, profile.Repository, name),
		Tag:     "latest",
	}

	body, err := cli.ImagePull(ctx, pullOpts, nil)
	if err != nil {
		isolate.GetLogger(ctx).WithError(err).WithFields(
			log.Fields{"name": name, "endpoint": profile.Endpoint,
				"image": pullOpts.ImageID, "tag": pullOpts.Tag}).Error("unable to pull an image")
		return err
	}
	defer body.Close()

	var (
		resp   spoolResponseProtocol
		logger = isolate.GetLogger(ctx).WithFields(log.Fields{"name": name, "endpoint": profile.Endpoint})
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
