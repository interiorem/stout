package porto

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	apexctx "github.com/m0sth8/context"
	"golang.org/x/net/context"

	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"

	"github.com/noxiouz/stout/isolate/docker"
)

type container struct {
	ctx context.Context

	containerID    string
	rootDir        string
	volumePath     string
	cleanupEnabled bool
}

type execInfo struct {
	*docker.Profile
	name, executable string
	args, env        map[string]string
}

type containerConfig struct {
	Root           string
	ID             string
	Layer          string
	CleanupEnabled bool
}

func formatCommand(executable string, args map[string]string) string {
	var buff = newBuff()
	defer buffPool.Put(buff)
	buff.WriteString(executable)
	for k, v := range args {
		buff.WriteByte(' ')
		buff.WriteString(k)
		buff.WriteByte(' ')
		buff.WriteString(v)
	}

	return buff.String()
}

func formatEnv(env map[string]string) string {
	var buff = newBuff()
	defer buffPool.Put(buff)
	for k, v := range env {
		buff.WriteString(k)
		buff.WriteByte('=')
		buff.WriteString(v)
		buff.WriteByte(';')
	}

	return buff.String()
}

func pickNetwork(network string) string {
	// TODO: this function is useless now
	// but we have to add more mapping later
	switch network {
	case "inherited":
		return network
	default:
		return "inherited"
	}
}

// NOTE: is it better to have some kind of our own init inside Porto container to handle output?

func newContainer(ctx context.Context, portoConn porto.API, cfg containerConfig, info execInfo) (cnt *container, err error) {
	volumeProperties := map[string]string{
		"backend": "overlay",
		"layers":  cfg.Layer,
		"private": "cocaine-app",
	}

	volumePath := filepath.Join(cfg.Root, "volume")
	if err = os.MkdirAll(volumePath, 0775); err != nil {
		return nil, err
	}
	defer func(err *error) {
		if *err != nil {
			apexctx.GetLogger(ctx).WithField("container", cfg.ID).Infof("cleaunup unfinished container footprint due to error %v", *err)
			portoConn.UnlinkVolume(volumePath, cfg.ID)
			os.RemoveAll(volumePath)
		}
	}(&err)

	volumeDescription, err := portoConn.CreateVolume(volumePath, volumeProperties)
	if err != nil {
		if !isEqualPortoError(err, portorpc.EError_VolumeAlreadyExists) {
			apexctx.GetLogger(ctx).WithError(err).WithField("container", cfg.ID).Error("unable to create volume")
			return nil, err
		}
		apexctx.GetLogger(ctx).WithField("container", cfg.ID).Info("volume already exists")
	} else {
		apexctx.GetLogger(ctx).WithField("container", cfg.ID).Infof("created volume %v", volumeDescription)
	}

	if err = portoConn.Create(cfg.ID); err != nil {
		return nil, err
	}

	// NOTE: Porto cannot mount directories to symlinked dirs
	hostDir := info.args["--endpoint"]
	info.args["--endpoint"] = "/run/cocaine"
	bind := hostDir + " " + info.args["--endpoint"]
	if err = portoConn.SetProperty(cfg.ID, "bind", bind); err != nil {
		return nil, err
	}
	if err = portoConn.SetProperty(cfg.ID, "command", formatCommand(info.executable, info.args)); err != nil {
		return nil, err
	}
	if err = portoConn.SetProperty(cfg.ID, "env", formatEnv(info.env)); err != nil {
		return nil, err
	}
	if info.Cwd != "" {
		if err = portoConn.SetProperty(cfg.ID, "cwd", info.Cwd); err != nil {
			return nil, err
		}
	}
	if err = portoConn.SetProperty(cfg.ID, "net", pickNetwork(string(info.NetworkMode))); err != nil {
		return nil, err
	}
	if info.Resources.Memory != 0 {
		if err = portoConn.SetProperty(cfg.ID, "memory_limit", strconv.FormatInt(info.Resources.Memory, 10)); err != nil {
			return nil, err
		}
	}
	if err = portoConn.SetProperty(cfg.ID, "root", volumePath); err != nil {
		return nil, err
	}
	if err = portoConn.LinkVolume(volumePath, cfg.ID); err != nil {
		return nil, err
	}

	cnt = &container{
		ctx: ctx,

		containerID:    cfg.ID,
		rootDir:        cfg.Root,
		volumePath:     volumePath,
		cleanupEnabled: cfg.CleanupEnabled,
	}
	return cnt, nil
}

func (c *container) start(portoConn porto.API, output io.Writer) (err error) {
	defer apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Trace("start container").Stop(&err)
	return portoConn.Start(c.containerID)
}

func (c *container) Kill() (err error) {
	defer apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Trace("Kill container").Stop(&err)
	containersKilledCounter.Inc(1)
	portoConn, err := portoConnect()
	if err != nil {
		return err
	}
	defer portoConn.Close()
	defer c.Cleanup(portoConn)

	if err = portoConn.Kill(c.containerID, syscall.SIGKILL); err != nil {
		if !isEqualPortoError(err, portorpc.EError_InvalidState) {
			return err
		}

		return nil
	}

	apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Debugf("footprint %s", containerFootprint{
		portoConn:   portoConn,
		containerID: c.containerID,
	})

	if _, err = portoConn.Wait([]string{c.containerID}, 5*time.Second); err != nil {
		return err
	}

	return nil
}

func (c *container) Cleanup(portoConn porto.API) {
	if !c.cleanupEnabled {
		return
	}

	var err error
	if err = portoConn.UnlinkVolume(c.volumePath, c.containerID); err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).WithError(err).Warnf("Unlink volume %s", c.volumePath)
	}
	if err = portoConn.UnlinkVolume(c.volumePath, "/"); err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", "/").WithError(err).Warnf("Unlink volume %s", c.volumePath)
	}
	if err = portoConn.Destroy(c.containerID); err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).WithError(err).Warn("Destroy error")
	}
	if err = os.RemoveAll(c.rootDir); err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).WithError(err).Warnf("Remove dirs %s", c.rootDir)
	}
}
