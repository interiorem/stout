package isolate

import (
	"crypto"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pborman/uuid"
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

func layerImported(layer string, importedLayers []string) bool {
	i := sort.SearchStrings(importedLayers, layer)
	return i < len(importedLayers) && importedLayers[i] == layer
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

func isEqualPortoError(err error, expectedErrno portorpc.EError) bool {
	switch err := err.(type) {
	case (*porto.Error):
		return err.Errno == expectedErrno
	default:
		return false
	}
}

func createLayerInPorto(host, downloadPath, layer string, portoConn porto.API) error {
	download := true
	// TODO: don't download the same layer twice
	layerPath := path.Join(downloadPath, layer+".tar.gz")
	file, err := os.OpenFile(layerPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if !os.IsExist(err) {

			return err
		}
		log.WithField("layer", layer).Info("skip downloaded layer")
		download = false
	}

	if download {
		defer os.Remove(layerPath)
		defer file.Close()
		var (
			expectedSize int64
			// expectedChecksum string
			hashSummer hash.Hash
		)
		// fetch images metainfo
		imageJSONURL := fmt.Sprintf("http://%s/v1/images/%s/json", host, layer)
		log.Infof("imageJSONURL: %s", imageJSONURL)
		resp, err := http.Get(imageJSONURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			// checksumHeader := resp.Header.Get("X-Docker-Payload-Checksum")
			// switch {
			// case len(checksumHeader) == 0:
			// 	log.Warning("empty X-Docker-Payload-Checksum header")
			// case strings.HasPrefix(checksumHeader, "sha512:"):
			// 	expectedChecksum = strings.TrimPrefix(checksumHeader, "sha512:")
			// 	hashSummer = crypto.SHA512.New()
			// case strings.HasPrefix(checksumHeader, "sha256:"):
			// 	expectedChecksum = strings.TrimPrefix(checksumHeader, "sha256:")
			// 	hashSummer = crypto.SHA256.New()
			// case strings.HasPrefix(checksumHeader, "md5:"):
			// 	expectedChecksum = strings.TrimPrefix(checksumHeader, "md5:")
			// 	hashSummer = crypto.MD5.New()
			// default:
			// 	log.Warningf("unknown X-Docker-Payload-Checksum %s", checksumHeader)
			// }

			if expectedSize, err = strconv.ParseInt(resp.Header.Get("X-Docker-Size"), 10, 64); err != nil {
				log.Warningf("unable to parse X-Docker-Size %s: %v", resp.Header.Get("X-Docker-Size"), err)
			}
		default:
			log.Infof("bad response status: %s. skip checksum and size check", resp.Status)
		}

		layerURL := fmt.Sprintf("http://%s/v1/images/%s/layer", host, layer)
		log.Infof("layerUrl: %s", layerURL)
		resp, err = http.Get(layerURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
		default:
			return fmt.Errorf("unknown reply %s", resp.Status)
		}

		// NOTE: it means we did not find proper hashsum in images JSON
		if hashSummer == nil {
			hashSummer = crypto.MD5.New()
		}

		// NOTE: check, that it does not affect io.Copy performance
		var src = io.TeeReader(resp.Body, hashSummer)
		nn, err := io.Copy(file, src)
		if err != nil {
			return err
		}

		actualChecksum := fmt.Sprintf("%x", hashSummer.Sum(nil))
		log.WithFields(log.Fields{"layer": layer, "layerPath": layerPath,
			"size": nn, "checksum": actualChecksum}).Info("layer has been downloaded")

		if expectedSize > 0 {
			if expectedSize != nn {
				log.WithFields(log.Fields{"layer": layer,
					"layerPath": layerPath}).Errorf("invalid size. Expected %d, but %d", expectedSize, nn)
				return fmt.Errorf("imvalid downloaded image size %d != %d", nn, expectedSize)
			}
		} else {
			log.WithFields(log.Fields{"layer": layer,
				"layerPath": layerPath,
				"merge":     false}).Warning("size check is skipped")
		}
	}

	log.WithFields(log.Fields{"layer": layer, "layerPath": layerPath, "merge": false}).Info("import layer")
	err = portoConn.ImportLayer(layer, layerPath, false)
	if err != nil {
		if !isEqualPortoError(err, portorpc.EError_LayerAlreadyExists) {
			log.WithFields(log.Fields{"layer": layer, "error": err}).Error("unable to import layer")
			return err
		}
		log.WithField("layer", layer).Infof("skip an already existed layer")
	}
	return nil
}

// PortoIsolationConfig is simple configuration options
// for portoIsolation
type PortoIsolationConfig struct {
	// Name of the parent container
	RootNamespace string `json:"root"`
	// Path to the directory for temporary layers
	// downloaded from Registry
	Layers string `json:"layers"`
	// Path to build Porto volumes
	Volumes string `json:"volumes"`
}

type portoIsolation struct {
	// Temporary place to download layers
	layersCache string
	// Path where volumes are created
	volumesPath string
	// Name of Root container
	rootNamespace string

	mu         sync.RWMutex
	containers map[string]string

	properties []string
	data       []string
}

//NewPortoIsolation creates Isolation instance which uses Porto
func NewPortoIsolation(config *PortoIsolationConfig) (Isolation, error) {
	rootNamespace := config.RootNamespace
	cachePath := config.Layers
	volumesPath := config.Volumes

	portoConn, err := porto.Connect()
	if err != nil {
		return nil, err
	}
	defer portoConn.Close()

	verTag, verRevision, err := portoConn.GetVersion()
	if err != nil {
		return nil, err
	}
	log.Infof("Porto version: %s %s", verTag, verRevision)

	if len(rootNamespace) > 0 {
		// TODO: check vital properties of the parent container
		_, err = portoConn.GetProperty(rootNamespace, "isolate")
		if err != nil {
			return nil, err
		}
	}

	if err := dirExists(cachePath); err != nil {
		log.WithFields(log.Fields{
			"error": err, "path": cachePath}).Warning("layers path does not exist")

		if err := os.MkdirAll(cachePath, 0755); err != nil {
			log.WithFields(log.Fields{
				"error": err, "path": cachePath}).Error("unable to create layers directory")
			return nil, err
		}
	}

	if !path.IsAbs(volumesPath) {
		return nil, fmt.Errorf("volumesPath must absolute: %s", volumesPath)
	}
	if err := dirExists(volumesPath); err != nil {
		log.WithFields(log.Fields{
			"error": err, "path": volumesPath}).Warning("volumes path does not exist")

		if err := os.MkdirAll(volumesPath, 0755); err != nil {
			log.WithFields(log.Fields{
				"error": err, "path": volumesPath}).Error("unable to create volumes directory")
			return nil, err
		}
	}

	var dataItems = []string{}
	data, err := portoConn.Dlist()
	if err != nil {
		return nil, err
	}
	for _, item := range data {
		dataItems = append(dataItems, item.Name)
	}

	var propertyItems = []string{}
	properties, err := portoConn.Plist()
	if err != nil {
		return nil, err
	}
	for _, item := range properties {
		propertyItems = append(propertyItems, item.Name)
	}

	return &portoIsolation{
		layersCache:   cachePath,
		volumesPath:   volumesPath,
		rootNamespace: rootNamespace,

		// TODO: fill it from Porto
		// available containers
		containers: make(map[string]string),

		properties: propertyItems,
		data:       dataItems,
	}, nil
}

func (pi *portoIsolation) volumePathForApp(appname string) string {
	return path.Join(pi.volumesPath, appname)
}

func (pi *portoIsolation) logContainerFootprint(portoConn porto.API, containerID string) {
	if log.GetLevel() < log.DebugLevel {
		return
	}

	logger := log.WithField("container", containerID)

	footprintLength := len(pi.properties) + len(pi.data)
	if footprintLength == 0 {
		logger.Debug("No footprints for container")
		return
	}

	logger.Debug("Log container footprints")
	footprint := make(map[string]string, footprintLength)

	for _, property := range pi.properties {
		value, err := portoConn.GetProperty(containerID, property)
		if err != nil {
			logger.WithField("error", err).Warnf("unable to get property %s", property)
			continue
		}
		footprint[property] = value
	}

	for _, data := range pi.data {
		value, err := portoConn.GetData(containerID, data)
		if err != nil {
			logger.WithField("error", err).Warnf("unable to get data %s", data)
			continue
		}
		footprint[data] = value
	}

	if body, err := json.Marshal(footprint); err != nil {
		logger.Debugf("%v %+v", err, footprint)
	} else {
		logger.Debugf("%s", body)
	}

	// NOTE: read limited amount of lines in the future
	if stderrPath, ok := footprint["stderr_path"]; ok {
		if stderr, err := ioutil.ReadFile(stderrPath); err != nil {
			logger.WithField("error", err).Error("unable to read stderr")
		} else {
			logger.Debugf("STDERR: %s", stderr)
		}
	}
	if stdoutPath, ok := footprint["stdout_path"]; ok {
		if stdout, err := ioutil.ReadFile(stdoutPath); err != nil {
			logger.WithField("error", err).Error("unable to read stderr")
		} else {
			logger.Debugf("STDOUT: %s", stdout)
		}
	}
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

	if log.GetLevel() >= log.DebugLevel {
		log.Debugf("layers %s", strings.Join(layers, " "))
	}

	portoConn, err := porto.Connect()
	if err != nil {
		return err
	}
	defer portoConn.Close()

	importedLayers, err := portoConn.ListLayers()
	if err != nil {
		return err
	}
	sort.Strings(importedLayers)

	for _, layer := range layers {
		if layerImported(layer, importedLayers) {
			log.WithFields(log.Fields{
				"layer": layer, "image": imagename}).Info("layer is already imported")
			continue
		}

		err = createLayerInPorto(host, pi.layersCache, layer, portoConn)
		if err != nil {
			return err
		}
	}

	// Create volume
	volumeProperties := map[string]string{
		"backend": "overlay",
		"layers":  strings.Join(layers, ";"),
		"private": "cocaine-app",
	}

	log.Infof("%v", volumeProperties)

	volumePath := pi.volumePathForApp(appname)
	if err := os.MkdirAll(volumePath, 0775); err != nil {
		log.WithFields(log.Fields{
			"imagename": imagename, "error": err, "path": volumePath}).Error("unable to create a volume dir")
		return err
	}

	volumeDescription, err := portoConn.CreateVolume(volumePath, volumeProperties)
	if err != nil {
		if !isEqualPortoError(err, portorpc.EError_VolumeAlreadyExists) {
			log.WithFields(log.Fields{"imageid": imageid, "error": err}).Error("unable to create volume")
			return err
		}
		log.WithField("imageid", imageid).Info("volume already exists")
	} else {
		log.WithField("imageid", imageid).Infof("Created volume %v", volumeDescription)
	}

	// NOTE: create a meta container for all the workers of an app
	parentContainer := path.Join(pi.rootNamespace, appname)
	err = portoConn.Create(parentContainer)
	if err != nil {
		if !isEqualPortoError(err, portorpc.EError_ContainerAlreadyExists) {
			log.WithFields(log.Fields{"parent": parentContainer, "error": err}).Error("unable to create container")
			return err
		}
		log.WithField("parent", parentContainer).Info("parent container already exists")
	}

	// NOTE: Link a created volume to the parent (meta application) container. It's just a ref counter.
	// Also this volume has been already linked to a parent (root) namespace.
	if err := portoConn.LinkVolume(volumePath, parentContainer); err != nil {
		if !isEqualPortoError(err, portorpc.EError_VolumeAlreadyLinked) {
			log.WithFields(log.Fields{"parent": parentContainer, "error": err, "volume": volumePath}).Error("unable to link volume")
			return err
		}
	}

	// NOTE: transfer the ownership of the volume for application meta container.
	// Current links: `root_namespace` `root_namespace/app_meta_container_namespace`
	// So unlink this volume from `root_namespace`
	if pi.rootNamespace != "" {
		if err := portoConn.UnlinkVolume(volumePath, pi.rootNamespace); err != nil {
			if !isEqualPortoError(err, portorpc.EError_VolumeNotLinked) {
				log.WithFields(log.Fields{"root": pi.rootNamespace, "error": err, "volume": volumePath}).Error("unable to unlink volume")
				return err
			}
			log.WithFields(log.Fields{"root": pi.rootNamespace, "error": err, "volume": volumePath}).Warn("volume is not linked")
		}
	}

	log.WithField("appname", appname).Info("Preparing parent container")
	if err := portoConn.SetProperty(parentContainer, "isolate", "true"); err != nil {
		log.WithField("appname", appname).Warnf("unable to set `isolate` property: %v", err)
	}

	return nil
}

func (pi *portoIsolation) Create(ctx context.Context, profile Profile) (salt string, err error) {
	image := profile.Image
	portoConn, err := porto.Connect()
	if err != nil {
		return "", err
	}
	defer portoConn.Close()

	_, appname := splitHostImagename(image)
	// TODO: check existance of the directory
	volumePath := pi.volumePathForApp(appname)
	log.WithField("app", appname).Info("generate container id for an application")
	salt = uuid.New()
	containerID := path.Join(pi.rootNamespace, appname, salt)

	log.WithFields(log.Fields{"containerID": containerID, "app": appname, "salt": salt}).Info("generated container id")
	if err := portoConn.Create(containerID); err != nil {
		return "", err
	}

	pi.mu.Lock()
	pi.containers[salt] = containerID
	pi.mu.Unlock()

	// NOTE: It's better to destroy a container if something goes wrong
	// TODO: wrap into ScopeExit
	defer func(containeID string) {
		if err != nil {
			log.WithField("container", containerID).Info("destroy container")
			if err := portoConn.Destroy(containerID); err != nil {
				log.WithFields(log.Fields{"container": containerID, "error": err}).Warning("unable to destroy container")
			}

			pi.mu.Lock()
			delete(pi.containers, salt)
			pi.mu.Unlock()
		}
	}(containerID)

	if log.GetLevel() <= log.DebugLevel {
		log.WithFields(log.Fields{"container": containerID, "command": profile.Command, "cwd": profile.WorkingDir,
			"net": profile.NetworkMode, "bind": profile.Bind, "root": volumePath}).Debug("set the properties explicitly")
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
	return salt, nil
}

func (pi *portoIsolation) Start(ctx context.Context, container string) error {
	portoConn, err := porto.Connect()
	if err != nil {
		return err
	}
	defer portoConn.Close()

	pi.mu.RLock()
	containerID, ok := pi.containers[container]
	pi.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no such container %s", container)
	}

	if err := portoConn.Start(containerID); err != nil {
		log.WithField("container", container).Errorf("unable to start container: %v", err)
	}

	return nil
}

func (pi *portoIsolation) Output(ctx context.Context, container string) (io.ReadCloser, error) {
	portoConn, err := porto.Connect()
	if err != nil {
		return nil, err
	}
	defer portoConn.Close()

	pi.mu.RLock()
	containerID, ok := pi.containers[container]
	pi.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no such container %s", container)
	}

	stdErrFile, err := portoConn.GetProperty(containerID, "stdout_path")
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

	pi.mu.Lock()
	containerID, ok := pi.containers[container]
	delete(pi.containers, container)
	pi.mu.Unlock()

	if !ok {
		return fmt.Errorf("no such container %s", container)
	}

	defer func() {
		pi.logContainerFootprint(portoConn, containerID)
		portoConn.Destroy(containerID)
	}()

	if err := portoConn.Kill(containerID, syscall.SIGTERM); err != nil {
		if !isEqualPortoError(err, portorpc.EError_InvalidState) {
			return err
		}

		return nil
	}

	defer func() {
		stopped, err := portoConn.Wait([]string{containerID}, time.Second*5)
		if err != nil {
			portoConn.Kill(containerID, syscall.SIGTERM)
			return
		}
		log.Debug(stopped)
		return
	}()

	return nil
}
