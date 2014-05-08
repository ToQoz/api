package jsonapi

import (
	"encoding/json"
	"fmt"
	"github.com/ToQoz/dou"
	"log"
	"net/http"
	"strconv"
)

// Register this plugin as "jsonapi"
func init() {
	dou.Register("jsonapi", &jsonAPI{})
}

type jsonAPI struct{}

// BeforeDispatch is default func for before dispatch.
func (ja *jsonAPI) BeforeDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return w, r
}

// AfterDispatch is default func for after dispatch.
func (ja *jsonAPI) AfterDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	return w, r
}

// OnPanic is called when panic occur.
func (ja *jsonAPI) OnPanic(w http.ResponseWriter, r *http.Request) {
	// if api.SafeWriter.Write called before occuring panic,
	// this will not write response body and header.
	// Because it is meaningless and foolish that jsonplugin.OnPanic break response body.
	// Example: Write([]byte("{}") -> some proccess -> panic -> jsonplugin.OnPanic -> Write([]byte(`{"message": "Internal Server Error"}`))
	//          -> Response body is {}{"message": "Internal Server Error"}.
	if sw, ok := w.(*dou.SafeWriter); ok {
		if sw.Wrote {
			return
		}
	}

	var b string

	j, err := json.Marshal(map[string]string{"message": http.StatusText(http.StatusInternalServerError)})

	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		b = http.StatusText(http.StatusInternalServerError)
	} else {
		b = string(j)
	}

	w.WriteHeader(http.StatusInternalServerError)

	_, err = fmt.Fprintln(w, b)

	if err != nil {
		// Skip error
		// http.Error skip this error too.
		log.Printf("dou: fail to fmt.Fpintln(http.ResponseWriter, string)\n%v", err)
	}
}

// Marshal a interface to a JSON.
func (ja *jsonAPI) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal JSON to a interface.
func (ja *jsonAPI) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// APIStatus sets code to X-API-Status header.
// X-API-Status means domestic application status.
// Sometimes api status can't be expressed only by http status.
// See http://blog.yappo.jp/yappo/archives/000829.html
func (ja *jsonAPI) APIStatus(w http.ResponseWriter, code int) {
	w.Header().Set("X-API-Status", strconv.Itoa(code))
}
