package dou

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var (
	plugins = map[string]Plugin{}
)

type Config map[string]interface{}

// Plugin is interface for dou.Api plugin.
// This plugin system make dou flexible and thin.
// You can create Plugin like a github.com/ToQoz/dou/jsonapi in accordance the use.
// see also github.com/ToQoz/dou/jsonapi
type Plugin interface {
	Recover(w http.ResponseWriter, r *http.Request)
	BeforeDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request)
	AfterDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request)
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
	ApiStatus(w http.ResponseWriter, code int)
}

// SafeWriter is safe http.ResponseWriter
// For prevent unintentionally multiple calling http.ResponseWriter.Write, this has bool `Worte`.
// When recovering panic, this is useful for prevent unintentionally writing to the continuation that was written before panic.
// Example: Write([]byte(`[]`)) ->
//          SomeFunc() -> (panic) -> Recover() ->
//          Write(`{"message": "Internal server erorr"}`)
// In ideal theory that I think, we have to prevent panic after calling Write. But no accident, no life :)
type SafeWriter struct {
	Wrote bool
	http.ResponseWriter
}

func NewSafeWriter(w http.ResponseWriter) *SafeWriter {
	return &SafeWriter{false, w}
}

func (sw *SafeWriter) Write(p []byte) (int, error) {
	sw.Wrote = true
	return sw.ResponseWriter.Write(p)
}

// Api is the bone of dou.
// Api adds a few triggers to http.Handler and provide a few useful helpers for creating api.
// Thanks of plugin system, Api don't need to be responsible for many compatible content-type and api domain rule.
type Api struct {
	Handler        http.Handler
	Config         Config
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	Listener       net.Listener
	LogStackTrace  bool
	plugin         Plugin
}

// Register makes a database driver available by the provided name.
func Register(pluginName string, plugin Plugin) {
	if plugin == nil {
		panic("github.com/ToQoz/dou: Register plugin is nil")
	}

	if _, dup := plugins[pluginName]; dup {
		panic("github.com/ToQoz/dou: Register called twice for plugin " + pluginName)
	}

	plugins[pluginName] = plugin
}

// Deregister plgugin that registered by the provided name.
func Deregister(pluginName string) {
	_, ok := plugins[pluginName]

	if !ok {
		log.Printf("github.com/ToQoz/dou: plugin %v is not registered. Can't deregister", pluginName)
		return
	}

	delete(plugins, pluginName)
}

// NewApi new and initialize Api.
func NewApi(pluginName string) (*Api, error) {
	plugin, ok := plugins[pluginName]

	if !ok {
		return nil, fmt.Errorf("github.com/ToQoz/dou: unknown plugin %q (forgotten import?)", pluginName)
	}

	api := new(Api)
	api.Config = Config{}
	api.plugin = plugin
	api.LogStackTrace = true

	return api, nil
}

func NewApiWithHandler(pluginName string, handler http.Handler) (*Api, error) {
	api, err := NewApi(pluginName)

	if err != nil {
		return nil, err
	}

	api.Handler = handler
	return api, nil
}

// ServeHTTP calls
//     1. call plugin.BeforeDispatch()
//     2. call plugin.ServeHTTP()
//     3. call plugin.AfterDispatch()
// And call plugin.Recover when panic occur.
// if panic occur before calling Api.plugin.AfterDispatch, this call it after recovering.
func (api *Api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sw := NewSafeWriter(w)

	// Recover if occur panic in Api.plugin.BeforeDispatch or plugin.Router.ServeHTTP
	recoverFunc := func() {
		if recv := recover(); recv != nil {
			if api.LogStackTrace {
				stacktrace := make([]byte, 2048)
				runtime.Stack(stacktrace, false)

				log.Printf("github.com/ToQoz/dou: Recover panic in plugin.ServeHTTP: %s\n%s", recv, stacktrace)
			}

			api.plugin.Recover(sw, r)
		}
	}

	// Even if panic occur in Api.plugin.BeforeDispatch or Api.Handler.ServerHTTP,
	// Api.plugin.AfterDispatch should be called.
	// Because we sometimes use Api.plugin.AfterDispatch as cleauper,
	// and response body is created before calling Api.plugin.AfterDispatch.

	func() {
		defer recoverFunc()
		w, r = api.plugin.BeforeDispatch(sw, r)
		api.Handler.ServeHTTP(w, r)
	}()

	func() {
		defer recoverFunc()
		api.plugin.AfterDispatch(w, r)
	}()
}

// ----------------------------------------------------------------------------
// ResponseWriter helpers `Ok/Error`
// ----------------------------------------------------------------------------

// Ok marshals and writes resource with http status code.
// Use this when you want to return non-error response.
func (api *Api) Ok(w http.ResponseWriter, resource interface{}, httpStatusCode int) {
	if httpStatusCode == 0 {
		httpStatusCode = http.StatusOK
	}

	b, err := api.Marshal(resource)

	if err != nil {
		// Unexpected error.
		// plugin.plugin.Recover will be called.
		panic(err)
	}

	w.WriteHeader(httpStatusCode)
	// Discard written size.
	// Because
	//     It returns the number of bytes written. If nn < len(p), it also returns an error explaining why the write is short.
	//     http://golang.org/pkg/bufio/#Writer.Write
	_, err = w.Write(b)

	if err != nil {
		// Skip this error.
		// http.Error skip too.
		// Only warn.
		log.Printf("github.com/ToQoz/dou: fail to http.ResponseWriter.Write([]byte)\n%v", err)
		return
	}
}

// Error marshals and writes resource with http status code.
// Use this when you want to return error response.
// This is almost same as api.Ok except NAME(Ok, Error).
func (api *Api) Error(w http.ResponseWriter, resource interface{}, httpStatusCode int) {
	if httpStatusCode == 0 {
		httpStatusCode = http.StatusInternalServerError
	}

	b, err := api.Marshal(resource)

	if err != nil {
		// Unexpected error.
		// plugin.plugin.Recover will be called.
		panic(err)
	}

	w.WriteHeader(httpStatusCode)
	// Discard written size
	// Because
	//     It returns the number of bytes written. If nn < len(p), it also returns an error explaining why the write is short.
	//     http://golang.org/pkg/bufio/#Writer.Write
	_, err = fmt.Fprintln(w, string(b))

	if err != nil {
		// Skip this error.
		// http.Error skip too.
		// Only warn.
		log.Printf("github.com/ToQoz/dou: fail to fmt.Fpintln(http.ResponseWriter, string)\n%v", err)
		return
	}
}

// ----------------------------------------------------------------------------
// Export plugin's func `ApiStatus/Marshal/Unmarshal`. They has possibility to be used from outside of Api.
// ----------------------------------------------------------------------------

// ApiStatus write api status code.
// It will be implemented by api.plugin.
// e.g. github.com/ToQoz/dou/jsonapi Use "X-API-Status" header.
func (api *Api) ApiStatus(w http.ResponseWriter, apiStatusCode int) {
	api.plugin.ApiStatus(w, apiStatusCode)
}

// Marshal encode v.
// Encoding procedure will be implemented by api.plugin
func (api *Api) Marshal(v interface{}) ([]byte, error) {
	return api.plugin.Marshal(v)
}

// Unarshal encode v.
// Decoding procedure will be implemented by api.plugin
func (api *Api) Unmarshal(data []byte, v interface{}) error {
	return api.plugin.Unmarshal(data, v)
}

// ----------------------------------------------------------------------------
// Server helpers `Run/Stop`
// ----------------------------------------------------------------------------

// Run api server.
// If fail to serve listener, output error and exit.
func (api *Api) Run(l net.Listener) {
	if api.Handler == nil {
		panic("github.com/ToQoz/dou: Api.Handler should not be nil")
	}

	api.Listener = l

	server := &http.Server{
		Handler:        api,
		ReadTimeout:    api.ReadTimeout,
		WriteTimeout:   api.WriteTimeout,
		MaxHeaderBytes: api.MaxHeaderBytes,
	}

	err := server.Serve(api.Listener)

	if err != nil {
		// skip error `http.errClosing`
		// Excuse:
		//     net.Listener.Close() closeing listener, but doesn't stop http.Server.Serve() loop.
		//     So net.Listener.Accept() in http.Server.Serve() return error `http.errClosing`.
		// This approach is unstable because of depending on not public error type but private error message.
		// But if it is changed, this occur panic. So we can notice.
		if strings.Contains(err.Error(), "use of closed network connection") {
			return
		}

		panic(err)
	}
}

// Stop api server
func (api *Api) Stop() {
	api.Listener.Close()
}
