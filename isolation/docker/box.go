package docker

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/noxiouz/stout/isolation"

	"github.com/apex/log"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

const (
	dockerVersionAPI = "v1.19"
)

var (
	defaultHeaders = map[string]string{"User-Agent": "cocaine-universal-isolation"}
)

type spoolResponseProtocol struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

// Box ...
type Box struct{}

// NewBox ...
func NewBox(cfg isolation.BoxConfig) (isolation.Box, error) {
	return &Box{}, nil
}

// Spawn spawns a prcess using container
func (b *Box) Spawn(ctx context.Context, opts isolation.Profile, name, executable string, args, env map[string]string) (isolation.Process, error) {
	return newContainer(ctx, Profile(opts), name, executable, args, env)
}

// Spool spools an image with a tag latest
func (b *Box) Spool(ctx context.Context, name string, opts isolation.Profile) (err error) {
	endpoint := Profile(opts).Endpoint()
	defer isolation.GetLogger(ctx).WithFields(log.Fields{"name": name, "endpoint": endpoint}).Trace("spooling an image").Stop(&err)

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
		logger = isolation.GetLogger(ctx).WithFields(log.Fields{"name": name, "endpoint": endpoint})
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
