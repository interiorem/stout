package docker

import (
	"bytes"
	"encoding/binary"

	"github.com/noxiouz/stout/isolate"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/docker/engine-api/types/strslice"

	"golang.org/x/net/context"
)

type process struct {
	ctx    context.Context
	output chan isolate.ProcessOutput

	dockerEndpoint string

	containerID string
}

func newContainer(ctx context.Context, profile Profile, name, executable string, args, env map[string]string) (proc isolate.Process, err error) {
	endpoint := profile.Endpoint()
	defer isolate.GetLogger(ctx).WithField("endpoint", endpoint).Trace("create container").Stop(&err)

	cli, err := client.NewClient(endpoint, dockerVersionAPI, nil, defaultHeaders)
	if err != nil {
		return nil, err
	}

	var image string
	if registry := profile.Registry(); registry != "" {
		image = registry + "/" + name
	} else {
		image = name
	}

	var Env = make([]string, 0, len(env))
	for k, v := range env {
		Env = append(Env, k+"="+v)
	}

	var Cmd = make(strslice.StrSlice, 1, len(args)+1)
	Cmd[0] = executable
	for k, v := range args {
		Cmd = append(Cmd, k, v)
	}

	var binds = make([]string, 1)
	binds[0] = args["--endpoint"] + ":" + profile.RuntimePath()

	config := container.Config{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,

		Env:        Env,
		Cmd:        Cmd,
		Image:      image,
		WorkingDir: "/",
	}

	hostConfig := container.HostConfig{
		NetworkMode: profile.NetworkMode(),
		Binds:       binds,
	}

	// NOTE: It should be nil
	var networkingConfig *network.NetworkingConfig

	resp, err := cli.ContainerCreate(ctx, &config, &hostConfig, networkingConfig, "")
	if err != nil {
		return nil, err
	}

	for _, warn := range resp.Warnings {
		isolate.GetLogger(ctx).Warnf("%s warning: %s", resp.ID, warn)
	}

	pr := &process{
		ctx:            ctx,
		output:         make(chan isolate.ProcessOutput, 10),
		containerID:    resp.ID,
		dockerEndpoint: endpoint,
	}

	if err := cli.ContainerStart(ctx, pr.containerID); err != nil {
		return nil, err
	}
	isolate.NotifyAbouStart(pr.output)
	go pr.collectOutput(cli)

	return pr, nil
}

func (p *process) Kill() (err error) {
	defer isolate.GetLogger(p.ctx).WithField("concontainer", p.containerID).Trace("Sending SIGKILL").Stop(&err)
	// Timeout?
	cli, err := client.NewClient(p.dockerEndpoint, dockerVersionAPI, nil, defaultHeaders)
	if err != nil {
		return err
	}

	defer func() {
		var err error
		defer isolate.GetLogger(p.ctx).WithField("concontainer", p.containerID).Trace("Removing a conatainer").Stop(&err)
		removeOpts := types.ContainerRemoveOptions{
			ContainerID: p.containerID,
		}

		err = cli.ContainerRemove(p.ctx, removeOpts)
	}()

	return cli.ContainerKill(p.ctx, p.containerID, "SIGKILL")
}

func (p *process) Output() <-chan isolate.ProcessOutput {
	return p.output
}

func (p *process) collectOutput(cli *client.Client) {
	attachOpts := types.ContainerAttachOptions{
		ContainerID: p.containerID,
		Stream:      true,
		Stdin:       false,
		Stdout:      true,
		Stderr:      true,
	}
	hjResp, err := cli.ContainerAttach(p.ctx, attachOpts)
	if err != nil {
		isolate.GetLogger(p.ctx).Infof("unable to attach to stdout/err of %s: %v", p.containerID, err)
		return
	}
	defer hjResp.Close()

	const headerSize = 8
	for {
		// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.22/#attach-a-container
		var header = make([]byte, headerSize)
		_, err := hjResp.Reader.Read(header)
		if err != nil {
			isolate.GetLogger(p.ctx).Infof("unable to read header for hjResp of %s: %v", p.containerID, err)
			select {
			case p.output <- isolate.ProcessOutput{Data: nil, Err: err}:
			case <-p.ctx.Done():
			}
			return

		}

		var size uint32
		if err = binary.Read(bytes.NewReader(header[3:]), binary.BigEndian, &size); err != nil {
			isolate.GetLogger(p.ctx).Infof("unable to decode szie from header of %s: %v", p.containerID, err)
			return
		}

		var output = make([]byte, size)
		_, err = hjResp.Reader.Read(output)
		if err != nil {
			isolate.GetLogger(p.ctx).Infof("unable to read output for hjResp of %s: %v", p.containerID, err)
			select {
			case p.output <- isolate.ProcessOutput{Data: nil, Err: err}:
			case <-p.ctx.Done():
			}
			return
		}

		select {
		case p.output <- isolate.ProcessOutput{Data: output, Err: nil}:
		case <-p.ctx.Done():
		}
	}
}
