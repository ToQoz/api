package api

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"
)

type Config map[string]interface{}
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

type Router interface {
	Get(string, http.Handler)
	Head(string, http.Handler)
	Post(string, http.Handler)
	Put(string, http.Handler)
	Delete(string, http.Handler)
	http.Handler
}

type APIError struct {
	ApiStatus int    `json:"api_status"`
	Message   string `json:"message"`
}

type APIErrors struct {
	ApiStatus int         `json:"api_status"`
	Errors    []*APIError `json:"errors"`
}

type Api struct {
	Router         Router
	Config         Config
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	Listener       net.Listener
	Server         *http.Server
}

func NewApi(router Router) *Api {
	api := &Api{Router: router, Config: Config{}}
	return api
}

// --- routing helper ---

func (api *Api) Get(path string, f HandlerFunc) {
	api.Router.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		f(w, r)
	}))
}

func (api *Api) Post(path string, f HandlerFunc) {
	api.Router.Post(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		f(w, r)
	}))
}

func (api *Api) Put(path string, f HandlerFunc) {
	api.Router.Put(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		f(w, r)
	}))
}

func (api *Api) Delete(path string, f HandlerFunc) {
	api.Router.Delete(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		f(w, r)
	}))
}

// --- error helper ---

// write `{message: "error content"}` with http-status-code:http.StatusInternalServerError
func (api *Api) Error(w http.ResponseWriter, err error) {
	api.ErrorWithHttpStatusAndApiStatus(w, err, http.StatusInternalServerError, 0)
}

// write `{message: "error content"}` with http-status-code
func (api *Api) ErrorWithHttpStatus(w http.ResponseWriter, err error, httpStatus int) {
	api.ErrorWithHttpStatusAndApiStatus(w, err, httpStatus, 0)
}

// write `{message: "error content"}` with http-status-code and api-status-code
func (api *Api) ErrorWithHttpStatusAndApiStatus(w http.ResponseWriter, err error, httpStatus, apiStatus int) {
	log.Print(err.Error())

	j, marchalError := json.Marshal(&APIError{Message: err.Error(), ApiStatus: apiStatus})

	if marchalError != nil {
		panic(marchalError)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	http.Error(w, string(j), httpStatus)
}

// write `{errors: [{message: "error content"}, {message: "error content"}]}` with http-status-code:http.StatusInternalServerError
func (api *Api) Errors(w http.ResponseWriter, errs []error) {
	api.ErrorsWithHttpStatusAndApiStatus(w, errs, http.StatusInternalServerError, 0)
}

// write `{errors: [{message: "error content"}, {message: "error content"}]}` with http-status-code
func (api *Api) ErrorsWithHttpStatus(w http.ResponseWriter, errs []error, httpStatus int) {
	api.ErrorsWithHttpStatusAndApiStatus(w, errs, httpStatus, 0)
}

// write `{errors: [{message: "error content"}, {message: "error content"}]}` with http-status-code and api-status-code
func (api *Api) ErrorsWithHttpStatusAndApiStatus(w http.ResponseWriter, errs []error, httpStatus, apiStatus int) {
	apiErrors := &APIErrors{ApiStatus: apiStatus}

	for _, err := range errs {
		log.Print(err.Error())
		apiErrors.Errors = append(apiErrors.Errors, &APIError{Message: err.Error()})
	}

	j, marchalError := json.Marshal(apiErrors)

	if marchalError != nil {
		panic(marchalError)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	http.Error(w, string(j), httpStatus)
}

// --- server helper ---

func (api *Api) Run(l net.Listener) {
	var err error

	api.Listener = l
	api.Server = &http.Server{
		Handler:        api.Router,
		ReadTimeout:    api.ReadTimeout,
		WriteTimeout:   api.WriteTimeout,
		MaxHeaderBytes: api.MaxHeaderBytes,
	}

	if err != nil {
		log.Fatalf("Could not listen: %s", api.Listener)
	}

	log.Printf("HTTP Server: %s", api.Listener.Addr())

	// Serve
	log.Fatalf("Error in Serve: %s", api.Server.Serve(api.Listener))
}

func (api *Api) Stop() {
	api.Listener.Close()
}
