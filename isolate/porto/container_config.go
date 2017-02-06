package porto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apexctx "github.com/m0sth8/context"
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

	v.linked = true
	return nil
}

func (v *portoVolume) Path() string {
	return v.path
}

func (v *portoVolume) Destroy(ctx context.Context, portoConn porto.API) error {
	log := apexctx.GetLogger(ctx).WithField("container", v.cID)
	var err error
	if v.linked {
		if err = portoConn.UnlinkVolume(v.path, v.cID); err != nil {
			log.WithError(err).Error("unlinking failed")
		} else {
			log.Debugf("volume %s successfully unlinked", v.path)
		}
		if err = portoConn.UnlinkVolume(v.path, "self"); err != nil {
			log.WithError(err).Error("unlinking from 'self' failed")
		} else {
			log.Debugf("volume %s successfully unlinked", v.path)
		}
	}
	if err = os.RemoveAll(v.path); err != nil {
		apexctx.GetLogger(ctx).WithError(err).WithField("container", v.cID).Error("remove root volume failed")
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
			apexctx.GetLogger(ctx).WithError(zerr).WithField("container", s.portoVolume.cID).Error("remove root volume failed")
		}
	}

	return err
}

type execInfo struct {
	*Profile
	name, executable, ulimits string
	args, env                 map[string]string
}

type containerConfig struct {
	execInfo

	Root           string
	ID             string
	Layer          string
	CleanupEnabled bool
	SetImgURI      bool
	VolumeBackend  string
}

func (c *containerConfig) CreateRootVolume(ctx context.Context, portoConn porto.API) (Volume, error) {
	properties := map[string]string{
		"backend": c.VolumeBackend,
		"layers":  c.Layer,
		"private": "cocaine-app",
	}

	log := apexctx.GetLogger(ctx).WithField("container", c.ID)
	for limit, value := range c.Profile.Volume {
		log.Debugf("apply volume limit %s %s", limit, value)
		properties[limit] = value
	}

	path := filepath.Join(c.Root, "volume")
	if err := os.MkdirAll(path, 0775); err != nil {
		return nil, err
	}

	log.Debugf("create porto root volume at %s with volumeProperties: %s", path, properties)
	volume := &portoVolume{
		cID:        c.ID,
		path:       path,
		properties: properties,
	}

	description, err := portoConn.CreateVolume(path, properties)
	if err != nil {
		log.WithError(err).Error("unable to create volume")
		volume.Destroy(ctx, portoConn)
		return nil, err
	}
	log.Debugf("porto volume has been created successfully %v", description)
	return volume, nil
}

func (c *containerConfig) CreateExtraVolumes(ctx context.Context, portoConn porto.API, root Volume) ([]Volume, error) {
	if len(c.Profile.ExtraVolumes) == 0 {
		return nil, nil
	}

	log := apexctx.GetLogger(ctx).WithField("container", c.ID)
	volumes := make([]Volume, 0, len(c.Profile.ExtraVolumes))

	cleanUpOnError := func() {
		for _, vol := range volumes {
			if err := vol.Destroy(ctx, portoConn); err != nil {
				log.WithError(err).Error("unable to clean up extra volume")
			} else {
				log.Info("volume has been cleaned up")
			}
		}
	}

	for _, volumeprofile := range c.Profile.ExtraVolumes {
		if volumeprofile.Target == "" {
			cleanUpOnError()
			return nil, fmt.Errorf("can not create volume with empty target")
		}

		path := filepath.Join(root.Path(), volumeprofile.Target)
		log.Debugf("create porto root volume at %s with volumeProperties: %s", path, volumeprofile.Properties)
		if err := os.MkdirAll(path, 0775); err != nil {
			log.WithError(err).Error("unable to create target directory")
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
			log.WithError(err).Error("unable to create extra volume")
			return nil, err
		}
		log.Debugf("extra volume has been created %v", description)
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

	// Options with merge policy: binds, env
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

	// Protected options: command, root, enable_porto, net (for now)
	properties["command"] = formatCommand(c.executable, c.args)
	properties["root"] = root.Path()
	properties["enable_porto"] = "false"
	properties["net"] = pickNetwork(c.NetworkMode)

	log := apexctx.GetLogger(ctx).WithField("container", c.ID)
	for property, value := range properties {
		log.Debugf("Set property %s %s", property, value)
		if err = portoConn.SetProperty(c.ID, property, value); err != nil {
			log.WithError(err).Errorf("SetProperty %s %s failed", property, value)
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
