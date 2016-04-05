package docker

import (
	"io/ioutil"

	"github.com/noxiouz/stout/isolation"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

const (
	dockerVersionAPI = "v1.19"
)

var (
	_ isolation.Box = &Box{}
)

// Box ...
type Box struct {
	ctx context.Context
}

// NewBox ...
func NewBox(cfg isolation.BoxConfig) (isolation.Box, error) {
	return nil, nil
}

// Spawn spawns a prcess using container
func (b *Box) Spawn(ctx context.Context, opts isolation.Profile, name, executable string, args, env map[string]string) (isolation.Process, error) {
	return nil, nil
}

// Spool spools an image with a tag latest
func (b *Box) Spool(ctx context.Context, name string, opts isolation.Profile) error {
	defaultHeaders := map[string]string{"User-Agent": "cocaine-universal-isolation"}
	cli, err := client.NewClient(client.DefaultDockerHost, dockerVersionAPI, nil, defaultHeaders)
	if err != nil {
		return err
	}

	pullOpts := types.ImagePullOptions{
		ImageID: name,
		// Tag:     "latest",
	}

	body, err := cli.ImagePull(b.ctx, pullOpts, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		isolation.GetLogger(ctx).Infof("Spool() read body error: %v", err)
		return err
	}

	isolation.GetLogger(ctx).Infof("%s", data)
	return nil
}
