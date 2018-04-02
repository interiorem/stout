package porto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/log"

	porto "github.com/yandex/porto/src/api/go"
	"golang.org/x/net/context"
)

type Volume interface {
	Link(ctx context.Context, portoConn porto.API) error
	Path() string
	Destroy(ctx context.Context, portoConn porto.API) error
}

type portoVolume struct {
	cID        string
	path       string
	linked     bool
	properties map[string]string
}

func (v *portoVolume) Link(ctx context.Context, portoConn porto.API) error {
	if err := portoConn.LinkVolume(v.path, v.cID); err != nil {
		return err
	}

	// try unlink new volume from root for protect from volume leak in some porto versions
	portoConn.UnlinkVolume(v.path, "/")

	v.linked = true
	return nil
}

func (v *portoVolume) Path() string {
	return v.path
}

func (v *portoVolume) Destroy(ctx context.Context, portoConn porto.API) error {
	lg := log.G(ctx).WithField("container", v.cID)
	var err error
	if v.linked {
		if err = portoConn.UnlinkVolume(v.path, v.cID); err != nil {
			lg.WithError(err).Error("unlinking failed")
		} else {
			lg.Debugf("volume %s successfully unlinked", v.path)
		}
		if err = portoConn.UnlinkVolume(v.path, "self"); err != nil {
			lg.WithError(err).Error("unlinking from 'self' failed")
		} else {
			lg.Debugf("volume %s successfully unlinked", v.path)
		}
	}

	// try unlink volume with linked = false from root at destroy phase
	if unlinkErr := portoConn.UnlinkVolume(v.path, "/"); unlinkErr != nil {
		lg.WithError(unlinkErr).Error("unlinking from '/' failed")
	} else {
		lg.Debugf("volume %s successfully unlinked from '/'", v.path)
	}

	if err = os.RemoveAll(v.path); err != nil {
		lg.WithError(err).WithField("container", v.cID).Error("remove root volume failed")
	}
	return err
}

type storageVolume struct {
	portoVolume

	storagepath string
}

func (s *storageVolume) Destroy(ctx context.Context, portoConn porto.API) error {
	err := s.portoVolume.Destroy(ctx, portoConn)

	if s.storagepath != "" {
		if zerr := os.RemoveAll(s.storagepath); zerr != nil {
			log.G(ctx).WithError(zerr).WithField("container", s.portoVolume.cID).Error("remove root volume failed")
		}
	}

	return err
}

type execInfo struct {
	*Profile
	name, executable, ulimits, resolv_conf string
	args, env                 map[string]string
}

type containerConfig struct {
	execInfo
	State           isolate.GlobalState

	Root            string
	ID              string
	Layer           string
	CleanupEnabled  bool
	SetImgURI       bool
	VolumeBackend   string
	Mtn             bool
	MtnAllocationId string
}

func (c *containerConfig) CreateRootVolume(ctx context.Context, portoConn porto.API) (Volume, error) {
	properties := map[string]string{
		"backend": c.VolumeBackend,
		"layers":  c.Layer,
		"private": "cocaine-app",
	}

	logger := log.G(ctx).WithField("container", c.ID)
	for limit, value := range c.Profile.Volume {
		logger.Debugf("apply volume limit %s %s", limit, value)
		properties[limit] = value
	}

	path := filepath.Join(c.Root, "volume")
	if err := os.MkdirAll(path, 0775); err != nil {
		return nil, err
	}

	logger.Debugf("create porto root volume at %s with volumeProperties: %s", path, properties)
	volume := &portoVolume{
		cID:        c.ID,
		path:       path,
		properties: properties,
	}

	description, err := portoConn.CreateVolume(path, properties)
	if err != nil {
		logger.WithError(err).Error("unable to create volume")
		volume.Destroy(ctx, portoConn)
		return nil, err
	}
	logger.Debugf("porto volume has been created successfully %v", description)
	return volume, nil
}

func (c *containerConfig) CreateExtraVolumes(ctx context.Context, portoConn porto.API, root Volume) ([]Volume, error) {
	if len(c.Profile.ExtraVolumes) == 0 {
		return nil, nil
	}

	logger := log.G(ctx).WithField("container", c.ID)
	volumes := make([]Volume, 0, len(c.Profile.ExtraVolumes))

	cleanUpOnError := func() {
		for _, vol := range volumes {
			if err := vol.Destroy(ctx, portoConn); err != nil {
				logger.WithError(err).Error("unable to clean up extra volume")
			} else {
				logger.Info("volume has been cleaned up")
			}
		}
	}

	for _, volumeprofile := range c.Profile.ExtraVolumes {
		if volumeprofile.Target == "" {
			cleanUpOnError()
			return nil, fmt.Errorf("can not create volume with empty target")
		}

		path := filepath.Join(root.Path(), volumeprofile.Target)
		logger.Debugf("create porto root volume at %s with volumeProperties: %s", path, volumeprofile.Properties)
		if err := os.MkdirAll(path, 0775); err != nil {
			logger.WithError(err).Error("unable to create target directory")
			cleanUpOnError()
			return nil, err
		}

		extraVolume := portoVolume{
			cID:  c.ID,
			path: path,

			properties: volumeprofile.Properties,
		}

		// There are 2 types of volumes we care about.
		// The first one requires storage directory for the data,
		// the second one does not (like tmpfs)
		if storage := volumeprofile.Properties["storage"]; storage != "" {
			// In case of storage type we wrap basic volume here
			storagevolume := storageVolume{
				portoVolume: extraVolume,
			}

			// NOTE: to make cleanUpOnError() clean even fresh container
			// storageVolume.Destroy handles situation with empty storagepath properly
			volumes = append(volumes, &storagevolume)

			storagepath := filepath.Join(storage, c.ID)
			if err := os.MkdirAll(storagepath, 0755); err != nil {
				cleanUpOnError()
				return nil, err
			}
			// storageVolume.Destroy will clean this directory
			storagevolume.storagepath = storagepath
			// rewrite relative storage path a real one
			volumeprofile.Properties["storage"] = storagepath
		} else {
			volumes = append(volumes, &extraVolume)
		}

		description, err := portoConn.CreateVolume(path, volumeprofile.Properties)
		if err != nil {
			cleanUpOnError()
			logger.WithError(err).Error("unable to create extra volume")
			return nil, err
		}
		log.G(ctx).Debugf("extra volume has been created %v", description)
	}

	return volumes, nil
}

func (c *containerConfig) CreateContainer(ctx context.Context, portoConn porto.API, root Volume, extraVolumes []Volume) (err error) {
	if err = portoConn.Create(c.ID); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			portoConn.Destroy(c.ID)
		}
	}()

	if c.SetImgURI {
		c.execInfo.env["image_uri"] = c.Profile.Registry + "/" + c.name
	}

	// As User can define arbitrary properties in `container` section,
	// some vital options like env, bind, command, root must be protected.
	// Depends on the option there different policies: merge, ro, etc.
	var properties = make(map[string]string, 7) // at least it has values

	// Unprotected values
	if c.ulimits != "" {
		properties["ulimit"] = c.ulimits
	}

	if c.Cwd != "" {
		properties["cwd"] = c.Cwd
	}

	for property, value := range c.Profile.Container {
		properties[property] = value
	}

	// Options with merge policy: binds, env. resolv_conf may be added from application profile in future.
	if env, ok := properties["env"]; ok {
		properties["env"] = env + ";" + formatEnv(c.env)
	} else {
		properties["env"] = formatEnv(c.env)
	}

	if binds, ok := properties["bind"]; ok {
		properties["bind"] = binds + ";" + formatBinds(&c.execInfo)
	} else {
		properties["bind"] = formatBinds(&c.execInfo)
	}

	if c.resolv_conf != "" {
		properties["resolv_conf"] = c.resolv_conf
	}

	// Protected options: command, root, enable_porto, net (for now)
	properties["command"] = formatCommand(c.executable, c.args)
	properties["root"] = root.Path()
	properties["enable_porto"] = "false"

	logger := log.G(ctx).WithField("container", c.ID)
	c.Mtn = false
	if !c.State.Mtn.Cfg.Enable {
		properties["net"] = pickNetwork(c.NetworkMode)
	} else {
		if c.Network["mtn"] == "enable" {
			alloc, err := c.State.Mtn.UseAlloc(ctx, string(c.Network["netid"]))
			if err != nil {
				logger.WithError(err).Errorf("get error from c.State.Mtn.UseAlloc, with netid: %s", c.Network["netid"])
				return err
			}
			properties["net"] = alloc.Net
			properties["hostname"] = alloc.Hostname
			properties["ip"] = alloc.Ip
			c.Mtn = true
			c.MtnAllocationId = alloc.Id
		}
	}

	//logger := log.G(ctx).WithField("container", c.ID)
	for property, value := range properties {
		logger.Debugf("Set property %s %s", property, value)
		if err = portoConn.SetProperty(c.ID, property, value); err != nil {
			logger.WithError(err).Errorf("SetProperty %s %s failed", property, value)
			return err
		}
	}

	if err = root.Link(ctx, portoConn); err != nil {
		portoConn.Destroy(c.ID)
		return err
	}

	for _, extraVolume := range extraVolumes {
		if err = extraVolume.Link(ctx, portoConn); err != nil {
			portoConn.Destroy(c.ID)
			return err
		}
	}

	return nil
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
	for _, dockerBind := range info.Profile.Binds {
		buff.WriteByte(';')
		buff.WriteString(strings.Replace(dockerBind, ":", " ", -2))
	}
	return buff.String()
}
