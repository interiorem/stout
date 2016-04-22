package docker

import (
	"bytes"
	"encoding/binary"
	"io"
	"path/filepath"

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
	client *client.Client
	output chan isolate.ProcessOutput

	containerID string
}

func newContainer(ctx context.Context, client *client.Client, profile *Profile, name, executable string, args, env map[string]string) (proc isolate.Process, err error) {
	defer isolate.GetLogger(ctx).Trace("spawning container").Stop(&err)

	var image string
	if registry := profile.Registry; registry != "" {
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
	binds[0] = filepath.Dir(args["--endpoint"]) + ":" + profile.RuntimePath

	config := container.Config{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,

		Env:        Env,
		Cmd:        Cmd,
		Image:      image,
		WorkingDir: profile.Cwd,
	}

	hostConfig := container.HostConfig{
		NetworkMode: profile.NetworkMode,
		Binds:       binds,
	}

	// NOTE: It should be nil
	var networkingConfig *network.NetworkingConfig

	resp, err := client.ContainerCreate(ctx, &config, &hostConfig, networkingConfig, "")
	if err != nil {
		isolate.GetLogger(ctx).WithError(err).Error("unable to create a container")
		return nil, err
	}

	for _, warn := range resp.Warnings {
		isolate.GetLogger(ctx).Warnf("%s warning: %s", resp.ID, warn)
	}

	pr := &process{
		ctx:         ctx,
		client:      client,
		output:      make(chan isolate.ProcessOutput, 10),
		containerID: resp.ID,
	}

	if err := client.ContainerStart(ctx, pr.containerID); err != nil {
		return nil, err
	}
	isolate.NotifyAbouStart(pr.output)
	go pr.collectOutput()

	return pr, nil
}

func (p *process) Kill() (err error) {
	defer isolate.GetLogger(p.ctx).WithField("container", p.containerID).Trace("Sending SIGKILL").Stop(&err)

	defer func() {
		var err error
		defer isolate.GetLogger(p.ctx).WithField("container", p.containerID).Trace("Removing a conatainer").Stop(&err)
		removeOpts := types.ContainerRemoveOptions{
			ContainerID: p.containerID,
		}

		err = p.client.ContainerRemove(p.ctx, removeOpts)
	}()

	return p.client.ContainerKill(p.ctx, p.containerID, "SIGKILL")
}

func (p *process) Output() <-chan isolate.ProcessOutput {
	return p.output
}

func (p *process) collectOutput() {
	defer close(p.output)

	attachOpts := types.ContainerAttachOptions{
		ContainerID: p.containerID,
		Stream:      true,
		Stdin:       false,
		Stdout:      true,
		Stderr:      true,
	}

	hjResp, err := p.client.ContainerAttach(p.ctx, attachOpts)
	if err != nil {
		isolate.GetLogger(p.ctx).WithError(err).Errorf("unable to attach to stdout/err of %s", p.containerID)
		return
	}
	defer hjResp.Close()

	const headerSize = 8
	for {
		// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.22/#attach-a-container
		/// NOTE: some logs can be lost because of EOF
		var header = make([]byte, headerSize)
		_, err := hjResp.Reader.Read(header)
		if err != nil {
			if err == io.EOF {
				return
			}

			isolate.GetLogger(p.ctx).WithError(err).Errorf("unable to read header for hjResp of %s", p.containerID)
			select {
			case p.output <- isolate.ProcessOutput{Data: nil, Err: err}:
			case <-p.ctx.Done():
			}
			return
		}

		var size uint32
		if err = binary.Read(bytes.NewReader(header[4:]), binary.BigEndian, &size); err != nil {
			isolate.GetLogger(p.ctx).WithError(err).Errorf("unable to decode size from header %s", p.containerID)
			return
		}

		var output = make([]byte, size)
		_, err = hjResp.Reader.Read(output)
		if err != nil {
			if err == io.EOF {
				return
			}

			isolate.GetLogger(p.ctx).WithError(err).Errorf("unable to read output for hjResp %s", p.containerID)
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
