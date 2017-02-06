package porto

import (
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

type rootVolume struct {
	cID        string
	path       string
	linked     bool
	properties map[string]string
}

func (v *rootVolume) Link(ctx context.Context, portoConn porto.API) error {
	if err := portoConn.LinkVolume(v.path, v.cID); err != nil {
		return err
	}

	v.linked = true
	return nil
}

func (v *rootVolume) Path() string {
	return v.path
}

func (v *rootVolume) Destroy(ctx context.Context, portoConn porto.API) error {
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

	log.Debugf("create porto volume at %s with volumeProperties: %s", path, properties)
	description, err := portoConn.CreateVolume(path, properties)
	if err != nil {
		log.WithError(err).Error("unable to create volume")
		os.RemoveAll(path)
		return nil, err
	}
	log.Infof("porto volume has been created successfully %v", description)

	volume := &rootVolume{
		cID:  c.ID,
		path: path,

		properties: properties,
	}
	return volume, nil
}

func (c *containerConfig) CreateExtraVolumes(ctx context.Context, portoConn porto.API, root Volume) ([]Volume, error) {

	return nil, nil
}

func (c *containerConfig) CreateContainer(ctx context.Context, portoConn porto.API, root Volume) (err error) {
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
