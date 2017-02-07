package porto

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/noxiouz/stout/pkg/log"
	"github.com/uber-go/zap"
	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

type container struct {
	ctx context.Context

	containerID    string
	rootDir        string
	cleanupEnabled bool
	SetImgURI      bool

	volume       Volume
	extraVolumes []Volume
	output       io.Writer
}

// NOTE: is it better to have some kind of our own init inside Porto container to handle output?
func newContainer(ctx context.Context, portoConn porto.API, cfg containerConfig) (cnt *container, err error) {
	lg := log.G(ctx).With(zap.String("container", cfg.ID))
	volume, err := cfg.CreateRootVolume(ctx, portoConn)
	if err != nil {
		lg.Error("root volume construction failed", zap.Error(err))
		return nil, err
	}

	extravolumes, err := cfg.CreateExtraVolumes(ctx, portoConn, volume)
	if err != nil {
		lg.Error("extra volumes construction failed")
		return nil, err
	}

	if err = cfg.CreateContainer(ctx, portoConn, volume, extravolumes); err != nil {
		volume.Destroy(ctx, portoConn)
		return nil, err
	}

	cnt = &container{
		ctx: ctx,

		containerID:    cfg.ID,
		rootDir:        cfg.Root,
		cleanupEnabled: cfg.CleanupEnabled,
		SetImgURI:      cfg.SetImgURI,

		volume:       volume,
		extraVolumes: extravolumes,
		output:       ioutil.Discard,
	}
	return cnt, nil
}

func (c *container) start(portoConn porto.API, output io.Writer) error {
	start := time.Now()
	c.output = output
	err := portoConn.Start(c.containerID)
	duration := time.Now().Sub(start)
	if err != nil {
		log.G(c.ctx).Error("failed to start container", zap.String("id", c.containerID), zap.Error(err), zap.Duration("duration", duration))
		return err
	}
	log.G(c.ctx).Info("start container successfully", zap.String("id", c.containerID), zap.Duration("duration", duration))
	return nil
}

func (c *container) Kill() (err error) {
	lg := log.G(c.ctx).With(zap.String("id", c.containerID))
	lg.Info("kill container")
	defer func(t time.Time) {
		duration := time.Now().Sub(t)
		if err != nil {
			lg.Error("failed to kill the container", zap.Error(err), zap.Duration("duration", duration))
			return
		}
		lg.Info("container successfully killed", zap.Duration("duration", duration))
	}(time.Now())

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
		lg.Warn("unable to get stdout", zap.Error(err))
	}
	// TODO: add StringWriter interface to an output
	c.output.Write([]byte(value))
	lg.Info("stdout has been sent", zap.Int("size", len(value)))

	value, err = portoConn.GetData(c.containerID, "stderr")
	if err != nil {
		lg.Warn("unable to get stderr", zap.Error(err))
	}
	c.output.Write([]byte(value))
	lg.Info("stderr has been sent", zap.Int("size", len(value)))

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
	lg := log.G(c.ctx).With(zap.String("id", c.containerID))

	var err error
	if err = c.volume.Destroy(c.ctx, portoConn); err != nil {
		lg.Warn("root volume has not been destroyed", zap.Error(err))
	} else {
		lg.Debug("root volume successfully destroyed")
	}

	for i, extraVolume := range c.extraVolumes {
		if err = extraVolume.Destroy(c.ctx, portoConn); err != nil {
			lg.Warn("extra volume has not been destroyed", zap.Error(err), zap.Int("num", i))
		} else {
			lg.Debug("extra volume successfully destroyed", zap.Int("num", i))
		}
	}
	if err = portoConn.Destroy(c.containerID); err != nil {
		lg.Warn("failed to destroy container", zap.Error(err))
	} else {
		lg.Debug("container successfully destroyed")
	}
	if err = os.RemoveAll(c.rootDir); err != nil {
		lg.Warn("remove dirs", zap.String("dirs", c.rootDir), zap.Error(err))
	} else {
		lg.Debug("remove dirs successfully", zap.String("dirs", c.rootDir))
	}
}
