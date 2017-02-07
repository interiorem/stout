package docker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"sync/atomic"

	"github.com/noxiouz/stout/isolate"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/docker/engine-api/types/strslice"

	apexctx "github.com/m0sth8/context"
	"context"
)

const (
	headerSize = 8

	// chunk size for logs
	chunkSize = 1024 * 1024
)

func containerRemove(client client.APIClient, ctx context.Context, id string) {
	var err error
	defer log.G(ctx).WithField("id", id).Trace("removing").Stop(&err)

	removeOpts := types.ContainerRemoveOptions{}
	err = client.ContainerRemove(ctx, id, removeOpts)
}

type process struct {
	ctx          context.Context
	cancellation context.CancelFunc

	client *client.Client

	containerID string

	removed uint32
}

func newContainer(ctx context.Context, client *client.Client, profile *Profile, name, executable string, args, env map[string]string) (pr *process, err error) {
	defer log.G(ctx).Trace("spawning container").Stop(&err)

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

	var binds = make([]string, 1, len(profile.Binds)+1)
	binds[0] = filepath.Dir(args["--endpoint"]) + ":" + profile.RuntimePath
	binds = append(binds, profile.Binds...)

	// update args["--endpoint"] according to the container's point of view
	args["--endpoint"] = filepath.Join(profile.RuntimePath, filepath.Base(args["--endpoint"]))

	var Cmd = make(strslice.StrSlice, 1, len(args)+1)
	Cmd[0] = executable
	for k, v := range args {
		Cmd = append(Cmd, k, v)
	}

	config := container.Config{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,

		Env:        Env,
		Cmd:        Cmd,
		Image:      image,
		WorkingDir: profile.Cwd,
		Labels:     map[string]string{isolateDockerLabel: name},
	}

	memorylimit, _ := profile.Resources.Memory.Int()
	cpuShares, _ := profile.Resources.CPUShares.Int()
	cpuPeriod, _ := profile.Resources.CPUPeriod.Int()
	cpuQuota, _ := profile.Resources.CPUQuota.Int()
	log.G(ctx).Info("applying Resource limits")
	var resources = container.Resources{
		Memory:     memorylimit,
		CPUShares:  cpuShares,
		CPUPeriod:  cpuPeriod,
		CPUQuota:   cpuQuota,
		CpusetCpus: profile.Resources.CpusetCpus,
		CpusetMems: profile.Resources.CpusetMems,
	}

	hostConfig := container.HostConfig{
		NetworkMode: container.NetworkMode(profile.NetworkMode),
		Binds:       binds,
		Resources:   resources,
	}

	if len(profile.Tmpfs) != 0 {
		buff := new(bytes.Buffer)
		for k, v := range profile.Tmpfs {
			fmt.Fprintf(buff, "%s: %s;", k, v)
		}
		log.G(ctx).Infof("mounting `tmpfs` to container: %s", buff.String())

		hostConfig.Tmpfs = profile.Tmpfs
	}

	// NOTE: It should be nil
	var networkingConfig *network.NetworkingConfig

	resp, err := client.ContainerCreate(ctx, &config, &hostConfig, networkingConfig, "")
	if err != nil {
		log.G(ctx).WithError(err).Error("unable to create a container")
		return nil, err
	}

	for _, warn := range resp.Warnings {
		log.G(ctx).Warnf("%s warning: %s", resp.ID, warn)
	}

	ctx, cancel := context.WithCancel(ctx)
	pr = &process{
		ctx:          ctx,
		cancellation: cancel,
		client:       client,
		containerID:  resp.ID,
	}

	return pr, nil
}

func (p *process) startContainer(wr io.Writer) error {
	var startBarier = make(chan struct{})
	go p.collectOutput(startBarier, wr)
	if err := p.client.ContainerStart(p.ctx, p.containerID, ""); err != nil {
		p.cancellation()
		return err
	}
	isolate.NotifyAboutStart(wr)
	close(startBarier)
	return nil
}

func (p *process) Kill() (err error) {
	defer log.G(p.ctx).WithField("id", p.containerID).Trace("Sending SIGKILL").Stop(&err)
	// release HTTP connections
	defer p.cancellation()
	defer p.remove()

	return p.client.ContainerKill(p.ctx, p.containerID, "SIGKILL")
}

func (p *process) remove() {
	if !atomic.CompareAndSwapUint32(&p.removed, 0, 1) {
		log.G(p.ctx).WithField("id", p.containerID).Info("already removed")
		return
	}
	containerRemove(p.client, p.ctx, p.containerID)
}

func (p *process) collectOutput(started chan struct{}, writer io.Writer) {
	attachOpts := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  false,
		Stdout: true,
		Stderr: true,
	}

	hjResp, err := p.client.ContainerAttach(p.ctx, p.containerID, attachOpts)
	if err != nil {
		log.G(p.ctx).WithError(err).Errorf("unable to attach to stdout/err of %s", p.containerID)
		return
	}
	defer hjResp.Close()

	var header = make([]byte, headerSize)
	for {
		// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.22/#attach-a-container
		/// NOTE: some logs can be lost because of EOF
		_, err := hjResp.Reader.Read(header)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.G(p.ctx).WithError(err).Errorf("unable to read header for hjResp of %s", p.containerID)
			return
		}

		var size uint32
		if err = binary.Read(bytes.NewReader(header[4:]), binary.BigEndian, &size); err != nil {
			log.G(p.ctx).WithError(err).Errorf("unable to decode size from header %s", p.containerID)
			return
		}

		if _, err = io.CopyN(writer, hjResp.Reader, int64(size)); err != nil {
			return
		}
	}
}
