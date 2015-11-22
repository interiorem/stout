package isolate

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"syscall"

	"code.google.com/p/go-uuid/uuid"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"

	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

func splitHostImagename(image string) (string, string) {
	index := strings.LastIndexByte(image, '/')
	return image[:index], image[index+1:]
}

func parseImageID(input io.Reader) (string, error) {
	body, err := ioutil.ReadAll(input)
	if err != nil {
		return "", err
	}

	imageid := string(body[1 : len(body)-1])
	return imageid, nil
}

func dirExists(path string) error {
	finfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !finfo.IsDir() {
		return fmt.Errorf("%s must be a directory", path)
	}

	return nil
}

func isItemExist(err error, expectedErrno portorpc.EError) bool {
	switch err := err.(type) {
	case (*porto.Error):
		return err.Errno == expectedErrno
	default:
		return false
	}
}

func createLayerInPorto(host, downloadPath, layer string, portoConn porto.API) error {
	layerPath := path.Join(downloadPath, layer+".tar.gz")
	file, err := os.OpenFile(layerPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if os.IsExist(err) {
			log.WithField("layer", layer).Info("skip downloaded layer")
			return nil
		}
		return err
	}
	defer os.Remove(layerPath)
	defer file.Close()

	layerURL := fmt.Sprintf("http://%s/v1/images/%s/layer", host, layer)
	log.Infof("layerUrl %s", layerURL)
	resp, err := http.Get(layerURL)
	if err != nil {
		file.Close()
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if _, err := io.Copy(file, resp.Body); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown reply %s", resp.Status)
	}

	err = portoConn.ImportLayer(layer, layerPath, false)
	if err != nil {
		if !isItemExist(err, portorpc.EError_LayerAlreadyExists) {
			log.WithFields(log.Fields{"layer": layer, "error": err}).Error("unbale to import layer")
			return err
		}
		log.WithField("layer", layer).Infof("skip an already existed layer")
	}
	return nil
}

type portoIsolation struct {
	// Temporary place to download layers
	layersCache string
	// Path where volumes are created
	volumesPath string
	// Name of Root container
	rootNamespace string
}

//NewPortoIsolation creates Isolation instance which uses Porto
func NewPortoIsolation() (Isolation, error) {
	cachePath := "/tmp/isolate"
	if err := dirExists(cachePath); err != nil {
		return nil, err
	}

	volumesPath := "/cocaine-porto"
	if !path.IsAbs(volumesPath) {
		return nil, fmt.Errorf("volumesPath must absolute: %s", volumesPath)
	}
	if err := dirExists(volumesPath); err != nil {
		return nil, err
	}

	return &portoIsolation{
		layersCache:   cachePath,
		volumesPath:   volumesPath,
		rootNamespace: "cocs",
	}, nil
}

func (pi *portoIsolation) volumePathForApp(appname string) string {
	return path.Join(pi.volumesPath, appname)
}

func (pi *portoIsolation) Spool(ctx context.Context, image, tag string) error {
	host, imagename := splitHostImagename(image)
	appname := imagename
	// get ImageId
	url := fmt.Sprintf("http://%s/v1/repositories/%s/tags/%s", host, imagename, tag)
	log.WithFields(log.Fields{
		"imagename": imagename, "tag": tag, "host": host, "url": url}).Info("fetching image id")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var imageid string
	switch resp.StatusCode {
	case http.StatusOK:
		if imageid, err = parseImageID(resp.Body); err != nil {
			log.WithField("error", err).Error("unable to parse image ID")
			return err
		}
	default:
		err := fmt.Errorf("invalid status code %s", resp.Status)
		log.WithField("error", err).Error("unable to fetch image id")
		return err
	}
	log.WithField("imagename", imagename).Infof("imageid has been fetched successfully")

	// get Ancestry
	ancestryURL := fmt.Sprintf("http://%s/v1/images/%s/ancestry", host, imageid)
	log.WithField("ancestryurl", ancestryURL).Info("fetching ancestry")
	resp, err = http.Get(ancestryURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var layers []string
	if err := json.NewDecoder(resp.Body).Decode(&layers); err != nil {
		return err
	}

	if len(layers) == 0 {
		return fmt.Errorf("an image without layers")
	}

	log.Debugf("layers %s", strings.Join(layers, " "))
	portoConn, err := porto.Connect()
	if err != nil {
		return err
	}
	defer portoConn.Close()

	// ImportLayers
	for _, layer := range layers {
		err := createLayerInPorto(host, pi.layersCache, layer, portoConn)
		if err != nil {
			return err
		}
	}

	// Create volume
	volumeProperties := map[string]string{
		"backend": "overlay",
		"layers":  strings.Join(layers, ";"),
		"private": "cocaine-app:" + imagename,
	}

	volumePath := pi.volumePathForApp(appname)
	if err := os.MkdirAll(volumePath, 0775); err != nil {
		log.WithFields(log.Fields{
			"imagename": imagename, "error": err, "path": volumePath}).Error("unable to create a volume dir")
		return err
	}

	volumeDescription, err := portoConn.CreateVolume(volumePath, volumeProperties)
	if err != nil {
		if !isItemExist(err, portorpc.EError_VolumeAlreadyExists) {
			log.WithFields(log.Fields{"imageid": imageid, "error": err}).Error("unable to create volume")
			return err
		}
		log.WithField("imageid", imageid).Info("volume already exists")
	} else {
		log.WithField("imageid", imageid).Infof("Created volume %v", volumeDescription)
	}

	// NOTE: create parent container
	parentContainer := path.Join(pi.rootNamespace, appname)
	err = portoConn.Create(parentContainer)
	if err != nil {
		if !isItemExist(err, portorpc.EError_ContainerAlreadyExists) {
			log.WithFields(log.Fields{"parent": parentContainer, "error": err}).Error("unable to create container")
			return err
		}
		log.WithField("parent", parentContainer).Info("parent container already exists")
	}

	// NOTE: it looks like a bug in Porto 2.6
	if err := portoConn.SetProperty(parentContainer, "isolate", "true"); err != nil {
		log.WithField("appname", appname).Warnf("unable to set `isolate` property: %v", err)
	}

	return nil
}

func (pi *portoIsolation) Create(ctx context.Context, profile Profile) (string, error) {
	image := profile.Image
	portoConn, err := porto.Connect()
	if err != nil {
		return "", err
	}
	defer portoConn.Close()

	// TODO: insert image nane in ID
	_, appname := splitHostImagename(image)
	// TODO: check existance of the directory
	volumePath := pi.volumePathForApp(appname)

	log.WithField("app", appname).Info("generate container id for an application")
	containerID := path.Join(pi.rootNamespace, appname, uuid.New())

	log.WithFields(log.Fields{"containerID": containerID, "app": appname}).Info("generated container id")
	if err := portoConn.Create(containerID); err != nil {
		return "", err
	}

	if err := portoConn.LinkVolume(volumePath, containerID); err != nil {
		log.Error(err)
		return "", err
	}

	if err := portoConn.SetProperty(containerID, "command", profile.Command); err != nil {
		return "", err
	}
	if err := portoConn.SetProperty(containerID, "cwd", profile.WorkingDir); err != nil {
		return "", err
	}
	if err := portoConn.SetProperty(containerID, "net", profile.NetworkMode); err != nil {
		return "", err
	}
	if err := portoConn.SetProperty(containerID, "bind", profile.Bind); err != nil {
		return "", err
	}
	if err := portoConn.SetProperty(containerID, "root", volumePath); err != nil {
		return "", err
	}
	return containerID, nil
}

func (pi *portoIsolation) Start(ctx context.Context, container string) error {
	portoConn, err := porto.Connect()
	if err != nil {
		return err
	}
	defer portoConn.Close()

	return portoConn.Start(container)
}

func (pi *portoIsolation) Output(ctx context.Context, container string) (io.ReadCloser, error) {
	portoConn, err := porto.Connect()
	if err != nil {
		return nil, err
	}
	defer portoConn.Close()

	stdErrFile, err := portoConn.GetProperty(container, "stdout_path")
	if err != nil {
		return nil, err
	}

	return os.Open(stdErrFile)
}

func (pi *portoIsolation) Terminate(ctx context.Context, container string) error {
	portoConn, err := porto.Connect()
	if err != nil {
		return err
	}
	defer portoConn.Close()
	defer portoConn.Destroy(container)

	if err := portoConn.Kill(container, syscall.SIGTERM); err != nil {
		return err
	}
	// TODO: add defer with Wait & syscall.SIGKILL

	return nil
}
