package docker

import (
	"bytes"
	"encoding/binary"

	"github.com/noxiouz/stout/isolation"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/docker/engine-api/types/strslice"

	"golang.org/x/net/context"
)

var (
	_ isolation.Process = &process{}

	defaultHeaders = map[string]string{"User-Agent": "cocaine-universal-isolation"}
)

type process struct {
	ctx    context.Context
	output chan isolation.ProcessOutput

	dockerEndpoint string

	containerID string
}

func newContainer(ctx context.Context, executable string, args, env map[string]string, workDir string) (isolation.Process, error) {
	dockerEndpoint := client.DefaultDockerHost
	cli, err := client.NewClient(dockerEndpoint, dockerVersionAPI, nil, defaultHeaders)
	if err != nil {
		return nil, err
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

	// TODO: pass opts
	var (
		config = container.Config{
			Env:        Env,
			Cmd:        Cmd,
			WorkingDir: workDir,

			AttachStdin:  false,
			AttachStdout: false,
			AttachStderr: false,
		}

		hostConfig = container.HostConfig{
			NetworkMode: "host",
		}

		// It should be nil
		networkingConfig *network.NetworkingConfig
	)

	resp, err := cli.ContainerCreate(ctx, &config, &hostConfig, networkingConfig, "")
	if err != nil {
		return nil, err
	}

	for _, warn := range resp.Warnings {
		isolation.GetLogger(ctx).Infof("%s warning: %s", resp.ID, warn)
	}

	pr := &process{
		ctx:            ctx,
		output:         make(chan isolation.ProcessOutput, 10),
		containerID:    resp.ID,
		dockerEndpoint: dockerEndpoint,
	}

	if err := cli.ContainerStart(ctx, pr.containerID); err != nil {
		return nil, err
	}
	isolation.NotifyAbouStart(pr.output)

	go func() {
		attachOpts := types.ContainerAttachOptions{
			ContainerID: pr.containerID,
			Stream:      true,
			Stdin:       false,
			Stdout:      true,
			Stderr:      true,
		}
		hjResp, err := cli.ContainerAttach(pr.ctx, attachOpts)
		if err != nil {
			isolation.GetLogger(ctx).Infof("unable to attach to stdout/err of %s: %v", pr.containerID, err)
			return
		}
		defer hjResp.Close()

		const headerSize = 8
		for {
			// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.22/#attach-a-container
			var header = make([]byte, headerSize)
			_, err := hjResp.Reader.Read(header)
			if err != nil {
				isolation.GetLogger(ctx).Infof("unable to read header for hjResp of %s: %v", pr.containerID, err)
				select {
				case pr.output <- isolation.ProcessOutput{Data: nil, Err: err}:
				case <-pr.ctx.Done():
				}
				return

			}

			var size uint32
			if err = binary.Read(bytes.NewReader(header[3:]), binary.BigEndian, &size); err != nil {
				isolation.GetLogger(ctx).Infof("unable to decode szie from header of %s: %v", pr.containerID, err)
				return
			}

			var output = make([]byte, size)
			_, err = hjResp.Reader.Read(output)
			if err != nil {
				isolation.GetLogger(ctx).Infof("unable to read output for hjResp of %s: %v", pr.containerID, err)
				select {
				case pr.output <- isolation.ProcessOutput{Data: nil, Err: err}:
				case <-pr.ctx.Done():
				}
				return
			}

			select {
			case pr.output <- isolation.ProcessOutput{Data: output, Err: nil}:
			case <-pr.ctx.Done():
			}
		}
	}()

	return pr, nil
}

func (p *process) Kill() error {
	// Timeout?
	cli, err := client.NewClient(p.dockerEndpoint, dockerVersionAPI, nil, defaultHeaders)
	if err != nil {
		return err
	}

	defer func() {
		removeOpts := types.ContainerRemoveOptions{
			ContainerID: p.containerID,
		}

		if err := cli.ContainerRemove(p.ctx, removeOpts); err != nil {
			isolation.GetLogger(p.ctx).Infof("ContainerRemove of container %s returns error: %v", p.containerID, err)
		} else {
			isolation.GetLogger(p.ctx).Infof("Conatiner %s has been removed successfully", p.containerID)
		}
	}()

	isolation.GetLogger(p.ctx).Infof("Send SIGKILL to stop container %s", p.containerID)

	return cli.ContainerKill(p.ctx, p.containerID, "SIGKILL")
}

func (p *process) Output() <-chan isolation.ProcessOutput {
	return p.output
}
