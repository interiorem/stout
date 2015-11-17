package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
)

func sendJSONError(w http.ResponseWriter, err error) {
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	})
}

func sendJSONContainerID(w http.ResponseWriter, containerID string) {
	json.NewEncoder(w).Encode(struct {
		ID string `json:"Id"`
	}{
		ID: containerID,
	})
}

//IsolateServer is a HTTP wrapper around PortoIsolation
type IsolateServer struct {
	*log.Entry

	Router *mux.Router
	isolate.Isolation
	ctx context.Context
}

type dockerPluginProfile struct {
	Image       string
	WorkingDir  string
	Cmd         []string
	NetworkMode string
	Env         []string
	Volumes     map[string]json.RawMessage
}

func (i *IsolateServer) spoolApplication(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-IsolateServer", "Porto")
	image := r.FormValue("fromImage")

	i.WithField("image", image).Info("spooling image")
	if err := i.Isolation.Spool(i.ctx, image, "latest"); err != nil {
		// Error is sent in body
		w.WriteHeader(http.StatusOK)
		sendJSONError(w, err)
		i.WithFields(log.Fields{
			"image": image, "error": err,
		}).Error("spooling failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "{}")
}

func (i *IsolateServer) containersCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-IsolateServer", "Porto")
	defer r.Body.Close()
	var profile dockerPluginProfile

	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	var binds []string
	for k := range profile.Volumes {
		binds = append(binds, k+" "+k)
	}

	portoProfile := isolate.Profile{
		Command: strings.Join(profile.Cmd, " "),
		//ToDO: add mapping
		NetworkMode: "host",
		Image:       profile.Image,
		WorkingDir:  profile.WorkingDir,
		Bind:        strings.Join(binds, ";"),
		Env:         strings.Join(profile.Env, ";") + ";",
	}

	container, err := i.Isolation.Create(i.ctx, portoProfile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		i.Errorf("unable to create container %v", err)
		return
	}

	sendJSONContainerID(w, container)
}

func (i *IsolateServer) containerAttach(w http.ResponseWriter, r *http.Request) {
	// ToDo: implement via hijacking
	w.Header().Add("X-IsolateServer", "Porto")
}

func (i *IsolateServer) containerDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-IsolateServer", "Porto")
	// Do nothing, just a placeholder
	w.WriteHeader(http.StatusOK)
}

func (i *IsolateServer) containerStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-IsolateServer", "Porto")
	container := mux.Vars(r)["container"]

	if err := i.Isolation.Start(i.ctx, container); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		i.Errorf("start container error: %v", err)
		fmt.Fprintf(w, "%v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (i *IsolateServer) containerKill(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-IsolateServer", "Porto")
	container := mux.Vars(r)["container"]

	if err := i.Isolation.Terminate(i.ctx, container); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		i.Errorf("kill container error: %v", err)
		fmt.Fprintf(w, "%v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (i *IsolateServer) fallback(w http.ResponseWriter, r *http.Request) {
	i.Errorf("fallback %s", r.URL.String())
	w.WriteHeader(http.StatusBadGateway)
}

//NewIsolateServer returns a HTTP wrapper around PortoIsolation
func NewIsolateServer() (*IsolateServer, error) {
	isolation, err := isolate.NewPortoIsolation()
	if err != nil {
		return nil, err
	}

	isolateServer := &IsolateServer{
		Entry: log.WithField("source", "server"),

		Router:    mux.NewRouter().Path("/{version}").Subrouter(),
		Isolation: isolation,
		ctx:       context.Background(),
	}

	isolateServer.Router.Path("/images/create").HandlerFunc(isolateServer.spoolApplication)
	isolateServer.Router.Path("/containers/create").HandlerFunc(isolateServer.containersCreate).Methods("POST")

	isolateServer.Router.Path("/containers/{container}/attach").HandlerFunc(isolateServer.containerAttach).Methods("POST")
	isolateServer.Router.Path("/containers/{container}/kill").HandlerFunc(isolateServer.containerKill).Methods("POST")
	isolateServer.Router.Path("/containers/{container}/start").HandlerFunc(isolateServer.containerStart).Methods("POST")
	isolateServer.Router.Path("/containers/{container}").HandlerFunc(isolateServer.containerDelete).Methods("DELETE")

	isolateServer.Router.Path("/").HandlerFunc(isolateServer.fallback)
	return isolateServer, nil
}
