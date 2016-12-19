package porto

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	apexctx "github.com/m0sth8/context"
	"golang.org/x/net/context"

	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

type container struct {
	ctx context.Context

	containerID    string
	rootDir        string
	volumePath     string
	cleanupEnabled bool
	SetImgURI      bool

	output io.Writer
}

type execInfo struct {
	portoProfile
	name, executable, ulimits string
	args, env                 map[string]string
}

type containerConfig struct {
	Root           string
	ID             string
	Layer          string
	CleanupEnabled bool
	SetImgURI      bool
	VolumeBackend  string
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
		if buff.Len() > 0 {
			buff.WriteByte(';')
		}
		buff.WriteString(k)
		buff.WriteByte('=')
		buff.WriteString(v)
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

// formatBinds prepares mount points for two cases:
// - endpoint with a cocaine socket. It always presents in info.args["--endpoint"]
// - optional mountpoints specified in the profile according to a Docker format
func formatBinds(info *execInfo) string {
	var buff = newBuff()
	defer buffPool.Put(buff)
	buff.WriteString(info.args["--endpoint"])
	buff.WriteByte(' ')
	buff.WriteString("/run/cocaine")
	info.args["--endpoint"] = "/run/cocaine"
	for _, dockerBind := range info.portoProfile.Binds() {
		buff.WriteByte(';')
		buff.WriteString(strings.Replace(dockerBind, ":", " ", -1))
	}
	return buff.String()
}

// NOTE: is it better to have some kind of our own init inside Porto container to handle output?

func newContainer(ctx context.Context, portoConn porto.API, cfg containerConfig, info execInfo) (cnt *container, err error) {
	apexctx.GetLogger(ctx).WithField("container", cfg.ID).Debugf("exec newContainer() with containerConfig: %s; execInfo: %s;", cfg, info)
	volumeProperties := map[string]string{
		"backend": cfg.VolumeBackend,
		"layers":  cfg.Layer,
		"private": "cocaine-app",
	}
	if err = info.applyVolumeLimits(ctx, cfg.ID, volumeProperties); err != nil {
		return nil, err
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
	apexctx.GetLogger(ctx).WithField("container", cfg.ID).Debugf("create volume with volumeProperties: %s", volumeProperties)
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

	if cfg.SetImgURI {
		info.env["image_uri"] = info.portoProfile.Registry() + "/" + info.name
	}

	if err = portoConn.SetProperty(cfg.ID, "bind", formatBinds(&info)); err != nil {
		return nil, err
	}
	if err = portoConn.SetProperty(cfg.ID, "command", formatCommand(info.executable, info.args)); err != nil {
		return nil, err
	}
	if err = portoConn.SetProperty(cfg.ID, "env", formatEnv(info.env)); err != nil {
		return nil, err
	}
	if info.ulimits != "" {
		if err = portoConn.SetProperty(cfg.ID, "ulimit", info.ulimits); err != nil {
			return nil, err
		}
	}
	if cwd := info.Cwd(); cwd != "" {
		if err = portoConn.SetProperty(cfg.ID, "cwd", cwd); err != nil {
			return nil, err
		}
	}
	if err = portoConn.SetProperty(cfg.ID, "net", pickNetwork(string(info.NetworkMode()))); err != nil {
		return nil, err
	}
	if err = portoConn.SetProperty(cfg.ID, "root", volumePath); err != nil {
		return nil, err
	}
	if err = portoConn.LinkVolume(volumePath, cfg.ID); err != nil {
		return nil, err
	}

	if err = info.portoProfile.applyContainerLimits(ctx, portoConn, cfg.ID); err != nil {
		return nil, err
	}

	cnt = &container{
		ctx: ctx,

		containerID:    cfg.ID,
		rootDir:        cfg.Root,
		volumePath:     volumePath,
		cleanupEnabled: cfg.CleanupEnabled,
		SetImgURI:      cfg.SetImgURI,

		output: ioutil.Discard,
	}
	return cnt, nil
}

func (c *container) start(portoConn porto.API, output io.Writer) (err error) {
	defer apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Trace("start container").Stop(&err)
	c.output = output
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

	// After Kill the container must be in `dead` state
	// Wait seems redundant as we sent SIGKILL
	value, err := portoConn.GetData(c.containerID, "stdout")
	if err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).WithError(err).Warn("unable to get stdout")
	}
	// TODO: add StringWriter interface to an output
	c.output.Write([]byte(value))
	apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Infof("%d bytes of stdout have been sent", len(value))

	value, err = portoConn.GetData(c.containerID, "stderr")
	if err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).WithError(err).Warn("unable to get stderr")
	}
	c.output.Write([]byte(value))
	apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Infof("%d bytes of stderr have been sent", len(value))

	if err = portoConn.Kill(c.containerID, syscall.SIGKILL); err != nil {
		if !isEqualPortoError(err, portorpc.EError_InvalidState) {
			return err
		}
		return nil
	}

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
	} else {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Debugf("Unlink volume %s successfully", c.volumePath)
	}
	if err = portoConn.UnlinkVolume(c.volumePath, "self"); err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", "self").WithError(err).Warnf("Unlink volume %s", c.volumePath)
	} else {
		apexctx.GetLogger(c.ctx).WithField("id", "self").Debugf("Unlink volume %s successfully", c.volumePath)
	}
	if err = portoConn.Destroy(c.containerID); err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).WithError(err).Warn("Destroy error")
	} else {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Debugf("Destroyed")
	}
	if err = os.RemoveAll(c.rootDir); err != nil {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).WithError(err).Warnf("Remove dirs %s", c.rootDir)
	} else {
		apexctx.GetLogger(c.ctx).WithField("id", c.containerID).Debugf("Remove dirs %s successfully", c.rootDir)
	}
}
