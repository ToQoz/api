package jsonapi

import (
	"bytes"
	"encoding/json"
	"github.com/ToQoz/dou"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BeforeDispatch should set Content-Type
func TestSetDefaultContentType(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a, err := dou.NewApiWithHandler("jsonapi", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	if err != nil {
		panic(err)
	}

	a.ServeHTTP(response, request)

	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Error("jsonapi.DefaultBeforeDispatch should set default content type `application/json; charset=utf-8`, but got %v", response.Header().Get("Content-Type"))
	}
}

// We can override Content-Type
func TestEnableToOverrideDefaultContentType(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a, err := dou.NewApiWithHandler("jsonapi", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
	}))

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	if err != nil {
		panic(err)
	}

	a.ServeHTTP(response, request)

	if response.Header().Get("Content-Type") != "text/plain" {
		t.Error("default content type that is set by jsonapi.DefaultBeforeDispatch should be overridable")
	}
}

// OnPanic should write response if panic occur
func TestOnPanicWriteErrorMessage(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a, err := dou.NewApiWithHandler("jsonapi", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("<test panic>")
	}))

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	if err != nil {
		panic(err)
	}

	a.ServeHTTP(response, request)

	if err != nil {
		panic(err)
	}

	gotContentType := response.Header().Get("Content-Type")

	switch {
	case strings.HasPrefix(gotContentType, "application/json;"):
		if string(response.Body.Bytes()) == "" {
			t.Error("OnPanic should write error message.")
		} else {

			gotJson := map[string]string{}
			err := json.Unmarshal(response.Body.Bytes(), &gotJson)

			if err != nil {
				panic(err)
			}

			if gotJson["message"] != http.StatusText(http.StatusInternalServerError) {
				t.Errorf("OnPanic wrote invalid error message. (got) = %s", response.Body.Bytes())
			}
		}
	case strings.HasPrefix(gotContentType, "text/plain;"):
		if string(response.Body.Bytes()) != http.StatusText(http.StatusInternalServerError) {
			t.Errorf("OnPanic wrote invalid error message. (expect) = %s, (got) = %s", http.StatusText(http.StatusInternalServerError), response.Body.Bytes())
		}
	default:
		t.Errorf("Unexpected Content-Type. (expect has prefix) = text/plain or application/json, (got) = %v", gotContentType)
	}
}

// OnPanic should not write if response is already written before panic occur
func TestOnPanicDontWriteIfResponseIsAlreadyWrittenBeforePanicOccur(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a, err := dou.NewApiWithHandler("jsonapi", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
		panic("<test panic>")
	}))

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	if err != nil {
		panic(err)
	}

	a.ServeHTTP(response, request)

	if !bytes.Equal(response.Body.Bytes(), []byte("hello")) {
		t.Error("OnPanic should not write response if response is written before panic occur")
	}
}

// ApiStatus should set X-API-Status header
func TestApiStatus(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a, err := dou.NewApi("jsonapi")
	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.ApiStatus(w, 999)
	})

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	if err != nil {
		panic(err)
	}

	a.ServeHTTP(response, request)

	if response.Header().Get("X-API-Status") != "999" {
		t.Errorf("ApiStatus should set X-API-Status. (expect) = \"999\", but (got) = %v", response.Header().Get("X-API-Status"))
	}
}
