package jsonapi

import (
	"encoding/json"
	"github.com/ToQoz/api"
	"net/http"
)

var (
	DefaultBeforeDispatch = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	DefaultAfterDispatch = func(w http.ResponseWriter, r *http.Request) {}
)

type apiError struct {
	ApiStatus int    `json:"api_status"`
	Message   string `json:"message"`
}

type apiErrors struct {
	ApiStatus int         `json:"api_status"`
	Errors    []*apiError `json:"errors"`
}

type jsonApi struct{}

func init() {
	api.Register("jsonapi", &jsonApi{})
}

func (ja *jsonApi) BeforeDispatch(w http.ResponseWriter, r *http.Request) {
	DefaultBeforeDispatch(w, r)
}

func (ja *jsonApi) AfterDispatch(w http.ResponseWriter, r *http.Request) {
	DefaultAfterDispatch(w, r)
}

// write `{"message": "<<err.Error()>>", "api_status": <<apiStatus>>}` with <<httpStatus>>
func (ja *jsonApi) Error(w http.ResponseWriter, err error, httpStatus, apiStatus int) {
	j, marchalError := json.Marshal(&apiError{Message: err.Error(), ApiStatus: apiStatus})

	if marchalError != nil {
		panic(marchalError)
	}

	http.Error(w, string(j), httpStatus)
}

// write `{"errors": [{"message": "<err.Error()>>"}, {"message": "<err.Error()>>"], api_status: <<apiStatus>>}` with <<httpStatus>>
func (ja *jsonApi) Errors(w http.ResponseWriter, errs []error, httpStatus, apiStatus int) {
	aErrs := &apiErrors{ApiStatus: apiStatus}

	for _, err := range errs {
		aErrs.Errors = append(aErrs.Errors, &apiError{Message: err.Error()})
	}

	j, marchalError := json.Marshal(aErrs)

	if marchalError != nil {
		panic(marchalError)
	}

	http.Error(w, string(j), httpStatus)
}
