package dou

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type recorder struct {
	calledTime int
}

func newRecorder() *recorder {
	return &recorder{calledTime: 0}
}

type testAPI struct {
	beforeDispatchCalled bool
	afterDispatchCalled  bool
	recoverCalled        bool
}

func (p *testAPI) BeforeDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	p.beforeDispatchCalled = true
	return w, r
}

func (p *testAPI) AfterDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	p.afterDispatchCalled = true
	return w, r
}

func (p *testAPI) OnPanic(w http.ResponseWriter, r *http.Request) {
	p.recoverCalled = true
}

func (p *testAPI) Marshal(v interface{}) ([]byte, error) {
	return nil, nil
}

func (p *testAPI) Unmarshal(data []byte, v interface{}) error {
	return nil
}

func (p *testAPI) APIStatus(w http.ResponseWriter, code int) {
}

func TestCallBeforeDispatchAndAfterDispatch(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	ta := &testAPI{}

	Register("testapi", ta)
	defer delete(plugins, "testapi")

	a, err := NewAPI("testapi")
	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	if err != nil {
		panic(err)
	}

	a.ServeHTTP(response, request)

	if !ta.beforeDispatchCalled {
		t.Error("Plugin.BeforeDispatch should be called")
	}

	if !ta.afterDispatchCalled {
		t.Error("Plugin.AfterDispatch should be called")
	}
}

func TestCallOnPanicIfOccurPanicInHandler(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	ta := &testAPI{}

	Register("testapi", ta)
	defer delete(plugins, "testapi")

	a, err := NewAPI("testapi")
	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("<test panic>")
	})

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	if err != nil {
		panic(err)
	}

	a.ServeHTTP(response, request)

	if !ta.recoverCalled {
		t.Error("Plugin.OnPanic should be called")
	}
}

func TestCallAfterDispatchIfOccurPanicInHandler(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	ta := &testAPI{}

	Register("testapi", ta)
	defer delete(plugins, "testapi")

	a, err := NewAPI("testapi")
	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("<test panic>")
	})

	if err != nil {
		panic(err)
	}

	a.LogStackTrace = false

	a.ServeHTTP(response, request)

	if !ta.afterDispatchCalled {
		t.Error("Plugin.AfterDispatch should be called")
	}
}

func TestNewSafeWriter(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := NewSafeWriter(w)

		if !reflect.DeepEqual(sw.ResponseWriter, w) {
			t.Error("NewSafeWriter should set given value to .ResponseWriter")
		}

		if sw.Wrote != false {
			t.Error("NewSafeWriter should set false to .wrote")
		}
	}).ServeHTTP(response, request)
}

func TestNewSafeWriterWrite(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := NewSafeWriter(w)
		sw.Write([]byte("hello"))

		if sw.Wrote != true {
			t.Error("NewSafeWriter.Wrote should be true after called Write")
		}
	}).ServeHTTP(response, request)
}
