package api

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type Config map[string]interface{}
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// Plugin interface. see also github.com/ToQoz/api/json/api
type Plugin interface {
	BeforeDispatch(w http.ResponseWriter, r *http.Request)
	AfterDispatch(w http.ResponseWriter, r *http.Request)
	Error(w http.ResponseWriter, err error, httpStatus, apiStatus int)
	Errors(w http.ResponseWriter, errs []error, httpStatus, apiStatus int)
}

// Router interface. You can use favorite router keeping this interface.
type Router interface {
	Get(string, http.Handler)
	Head(string, http.Handler)
	Post(string, http.Handler)
	Put(string, http.Handler)
	Delete(string, http.Handler)
	http.Handler
}

type Api struct {
	Router         Router
	Config         Config
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	Listener       net.Listener
	plugin         Plugin
}

var (
	plugins = map[string]Plugin{}
)

func Register(name string, p Plugin) {
	if p == nil {
		panic("api: Register plugin is nil")
	}

	if _, dup := plugins[name]; dup {
		panic("api: Register called twice for plugin " + name)
	}

	plugins[name] = p
}

func NewApi(pluginName string, router Router) (*Api, error) {
	plugin, ok := plugins[pluginName]

	if !ok {
		return nil, fmt.Errorf("api: unknown plugin %q (forgotten import?)", pluginName)
	}

	api := &Api{Router: router, Config: Config{}, plugin: plugin}
	return api, nil
}

// --- routing helper ---

func (api *Api) Get(path string, f HandlerFunc) {
	api.Router.Get(path, http.HandlerFunc(f))
}

func (api *Api) Post(path string, f HandlerFunc) {
	api.Router.Post(path, http.HandlerFunc(f))
}

func (api *Api) Put(path string, f HandlerFunc) {
	api.Router.Put(path, http.HandlerFunc(f))
}

func (api *Api) Delete(path string, f HandlerFunc) {
	api.Router.Delete(path, http.HandlerFunc(f))
}

func (api *Api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.plugin.BeforeDispatch(w, r)
	api.Router.ServeHTTP(w, r)
	api.plugin.AfterDispatch(w, r)
}

// --- error helper ---

func (api *Api) Error(w http.ResponseWriter, err error, httpStatus, apiStatus int) {
	api.plugin.Error(w, err, httpStatus, apiStatus)
}

func (api *Api) Errors(w http.ResponseWriter, errs []error, httpStatus, apiStatus int) {
	api.plugin.Errors(w, errs, httpStatus, apiStatus)
}

// --- server helper ---

func (api *Api) Run(l net.Listener) {
	var err error

	api.Listener = l
	server := &http.Server{
		Handler:        api,
		ReadTimeout:    api.ReadTimeout,
		WriteTimeout:   api.WriteTimeout,
		MaxHeaderBytes: api.MaxHeaderBytes,
	}

	if err != nil {
		log.Fatalf("Could not listen: %s", api.Listener.Addr())
	}

	log.Printf("HTTP Server: %s", api.Listener.Addr())

	// Serve
	log.Fatalf("Error in Serve: %s", server.Serve(api.Listener))
}

func (api *Api) Stop() {
	api.Listener.Close()
}
