package jsonapi

import (
	"encoding/json"
	"fmt"
	"github.com/ToQoz/dou"
	"log"
	"net/http"
	"strconv"
)

var (
	BeforeDispatchFunc = func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
		return DefaultBeforeDispatch(w, r)
	}
	AfterDispatchFunc = func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
		return DefaultAfterDispatch(w, r)
	}
	ApiStatus = func(w http.ResponseWriter, code int) {
		DefaultApiStatus(w, code)
	}
)

// Register this plugin as "jsonapi"
func init() {
	dou.Register("jsonapi", &jsonApi{})
}

func DefaultBeforeDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return w, r
}

func DefaultAfterDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	return w, r
}

// DefaultApiStatus sets code to X-API-Status header.
// X-API-Status means domestic application status.
// Sometimes api status can't be expressed only by http status.
// See http://blog.yappo.jp/yappo/archives/000829.html
func DefaultApiStatus(w http.ResponseWriter, code int) {
	w.Header().Set("X-API-Status", strconv.Itoa(code))
}

type jsonApi struct{}

// BeforeDispatch is called before dispatch.
// You change behavior by overriding jsonapi.BeforeDispatch.
func (ja *jsonApi) BeforeDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	if BeforeDispatchFunc == nil {
		return w, r
	}

	return BeforeDispatchFunc(w, r)
}

// BeforeDispatch is called after dispatch.
// You change behavior overriding jsonapi.AfterDispatch.
func (ja *jsonApi) AfterDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	if AfterDispatchFunc == nil {
		return w, r
	}

	return AfterDispatchFunc(w, r)
}

// Recover is called when panic occur.
func (ja *jsonApi) Recover(w http.ResponseWriter, r *http.Request) {
	// if api.SafeWriter.Write called before occuring panic,
	// this will not write response body and header.
	// Because it is meaningless and foolish that jsonplugin.Recover break response body.
	// Example: Write([]byte("{}") -> some proccess -> panic -> jsonplugin.Recover -> Write([]byte(`{"message": "Internal Server Error"}`))
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
func (ja *jsonApi) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal JSON to a interface.
func (ja *jsonApi) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ApiStatus sets domestic application status.
// By default, sets it to X-API-JSON header.
// You change it by overriding jsonapi.ApiStatus.
func (ja *jsonApi) ApiStatus(w http.ResponseWriter, code int) {
	if ApiStatus == nil {
		return
	}

	ApiStatus(w, code)
}
