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
)

type portoIsolation struct {
	// handlers    map[string]*portoHandler
	layersCache string
}

//NewPortoIsolation creates Isolation instance which uses Porto
func NewPortoIsolation() (Isolation, error) {
	return &portoIsolation{
		layersCache: "/tmp/isolate",
	}, nil
}

func (pi *portoIsolation) Spool(ctx context.Context, image, tag string) error {
	index := strings.LastIndexByte(image, '/')
	host, imagename := image[:index], image[index+1:]

	url := fmt.Sprintf("https://%s/v1/repositories/%s/tags/%s", host, imagename, tag)
	log.WithFields(log.Fields{
		"imagename": imagename,
		"tag":       tag,
		"host":      host}).Info("pull image  ", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	imageid := string(body[1 : len(body)-1])
	log.Infof("imageid %s", imageid)

	ancestryURL := fmt.Sprintf("https://%s/v1/images/%s/ancestry", host, imageid)
	log.WithField("ancestryurl", ancestryURL).Info("fetch ancestry")
	resp, err = http.Get(ancestryURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var layers []string
	if err := json.NewDecoder(resp.Body).Decode(&layers); err != nil {
		return err
	}
	log.Infof("layers %v", layers)
	for _, layer := range layers {
		layerPath := path.Join(pi.layersCache, layer+".tar.gz")
		file, err := os.OpenFile(layerPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			if os.IsExist(err) {
				log.Infof("Skip %s", layer)
				continue
			}
			return err
		}

		layerURL := fmt.Sprintf("https://%s/v1/images/%s/layer", host, layer)
		log.Infof("layerUrl %s", layerURL)
		resp, err = http.Get(layerURL)
		if err != nil {
			file.Close()
			return err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			if _, err := io.Copy(file, resp.Body); err != nil {
				file.Close()
				return err
			}
		default:
			file.Close()
			return fmt.Errorf("unknown reply %s", resp.Status)
		}
	}

	body, err = json.Marshal(layers)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(pi.layersCache, strings.Replace(image, "/", "__", -1)), body, 0666); err != nil {
		return err
	}
	return nil
}

func (pi *portoIsolation) Create(ctx context.Context, profile Profile) (string, error) {
	// Create contatiner
	// Start

	image := profile.Image
	log.Infof("Create layers for image %s", image)
	body, err := ioutil.ReadFile(path.Join(pi.layersCache, strings.Replace(image, "/", "__", -1)))
	if err != nil {
		return "", err
	}

	var layers []string
	if err := json.Unmarshal(body, &layers); err != nil {
		return "", err
	}

	portoConn, err := porto.NewPortoConnection()
	if err != nil {
		return "", err
	}
	defer portoConn.Close()

	// Create layers
	for _, layer := range layers {
		layerPath := path.Join(pi.layersCache, layer+".tar.gz")
		log.Infof("importing layer %s %s", layer, layerPath)

		if err := portoConn.ImportLayer(layer, layerPath, false); err != nil {
			if err.Error() == "LayerAlreadyExists" {
				log.Infof("layer %s already exists. Skip it", layer)
				continue
			}
			return "", err
		}
	}
	// Create volume
	volumeProperties := map[string]string{
		"backend": "overlay",
		"layers":  strings.Join(layers, ";"),
	}

	log.Infof("%v", volumeProperties)
	// path  must be empty for autogeneration
	volumeDescription, err := portoConn.CreateVolume("", volumeProperties)
	if err != nil {
		log.Errorf("unable to create volume %v", err)
		return "", err
	}

	log.Infof("%v", volumeDescription)
	// ToDo: insert image nane in ID
	containerID := uuid.New()
	log.Infof("containerId %s", containerID)
	if err := portoConn.Create(containerID); err != nil {
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
	if err := portoConn.SetProperty(containerID, "root", volumeDescription.Path); err != nil {
		return "", err
	}
	return containerID, nil
}

func (pi *portoIsolation) Start(ctx context.Context, container string) error {
	portoConn, err := porto.NewPortoConnection()
	if err != nil {
		return err
	}
	defer portoConn.Close()

	return portoConn.Start(container)
}

func (pi *portoIsolation) Output(ctx context.Context, container string) (io.ReadCloser, error) {
	portoConn, err := porto.NewPortoConnection()
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
	portoConn, err := porto.NewPortoConnection()
	if err != nil {
		return err
	}
	defer portoConn.Close()
	defer portoConn.Destroy(container)

	if err := portoConn.Kill(container, int32(syscall.SIGTERM)); err != nil {
		return err
	}

	return nil
}
