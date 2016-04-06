package docker

import (
	"bufio"
	"encoding/json"
	"fmt"

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
	return newContainer(ctx, Profile(opts), name, executable, args, env)
}

// Spool spools an image with a tag latest
func (b *Box) Spool(ctx context.Context, name string, opts isolate.Profile) (err error) {
	endpoint := Profile(opts).Endpoint()
	defer isolate.GetLogger(ctx).WithFields(log.Fields{"name": name, "endpoint": endpoint}).Trace("spooling an image").Stop(&err)

	cli, err := client.NewClient(endpoint, dockerVersionAPI, nil, defaultHeaders)
	if err != nil {
		return err
	}

	pullOpts := types.ImagePullOptions{
		ImageID: name,
		Tag:     "latest",
	}

	body, err := cli.ImagePull(ctx, pullOpts, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	var (
		resp   spoolResponseProtocol
		logger = isolate.GetLogger(ctx).WithFields(log.Fields{"name": name, "endpoint": endpoint})
	)

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		if err = json.Unmarshal(scanner.Bytes(), &resp); err != nil {
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
