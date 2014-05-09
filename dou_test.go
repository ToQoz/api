package dou

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

// Enable to stub
var testAPIMarshal = func(v interface{}) ([]byte, error) {
	return nil, nil
}

// Enable to stub
var testAPIUnmarshal = func(data []byte, v interface{}) error {
	return nil
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
	return testAPIMarshal(v)
}

func (p *testAPI) Unmarshal(data []byte, v interface{}) error {
	return testAPIUnmarshal(data, v)
}

func (p *testAPI) APIStatus(w http.ResponseWriter, code int) {
}

func TestNewAPIWithUnregisteredPluginName(t *testing.T) {
	api, err := NewAPI("unknown")

	if api != nil {
		t.Error("If NewAPI called with unknown plugin name, api should be nil")
	}

	if err == nil {
		t.Error("If NewAPI called with unknown plugin name, api should not be nil")
	}
}

func TestDeregister(t *testing.T) {
	Register("testapi", &testAPI{})
	Deregister("testapi")

	if _, ok := plugins["testapi"]; ok != false {
		t.Error("Deregister should delete registered plugin")
	}
}

func TestCallBeforeDispatchAndAfterDispatch(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	defer delete(plugins, "testapi")

	a := newTestAPI()
	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	a.LogStackTrace = false

	a.ServeHTTP(response, request)

	plugin, ok := a.Plugin.(*testAPI)

	if !ok {
		panic("a.plugin should be able to cast *testAPI")
	}

	if !plugin.beforeDispatchCalled {
		t.Error("Plugin.BeforeDispatch should be called")
	}

	if !plugin.afterDispatchCalled {
		t.Error("Plugin.AfterDispatch should be called")
	}
}

func TestCallOnPanicIfOccurPanicInHandler(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a := newTestAPI()
	a.LogStackTrace = false

	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("<test panic>")
	})

	a.ServeHTTP(response, request)

	plugin, ok := a.Plugin.(*testAPI)

	if !ok {
		panic("a.plugin should be able to cast *testAPI")
	}

	if !plugin.recoverCalled {
		t.Error("Plugin.OnPanic should be called")
	}
}

func TestCallAfterDispatchIfOccurPanicInHandler(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a := newTestAPI()
	a.LogStackTrace = false

	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("<test panic>")
	})

	a.ServeHTTP(response, request)

	plugin, ok := a.Plugin.(*testAPI)

	if !ok {
		panic("a.plugin should be able to cast *testAPI")
	}

	if !plugin.afterDispatchCalled {
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

func TestAPIUnmarshal(t *testing.T) {
	request, _ := http.NewRequest("GET", `/?json={"name": "ToQoz"}`, nil)
	response := httptest.NewRecorder()

	// stub testAPI.Marshal
	testAPIUnmarshal = func(data []byte, v interface{}) error {
		return json.Unmarshal(data, v)
	}

	a := newTestAPI()

	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := map[string]string{}

		expected := map[string]string{"name": "ToQoz"}

		a.Unmarshal([]byte(request.FormValue("json")), &v)

		if !reflect.DeepEqual(v, expected) {
			t.Errorf("fail to Unmarshal.\nexpected: %v\ngot: %v\n", expected, v)
		}
	})

	a.ServeHTTP(response, request)

}

func TestAPIOkSetGivenHTTPStatusCodeAndResponseBody(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	expectedBodyString := "stubed testAPIMarshal"
	expectedCode := http.StatusCreated

	// stub testAPI.Marshal
	testAPIMarshal = func(v interface{}) ([]byte, error) {
		return []byte(expectedBodyString), nil
	}

	a := newTestAPI()

	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.Ok(w, "", expectedCode)
	})

	a.ServeHTTP(response, request)

	// [Test] http status code
	if response.Code != expectedCode {
		t.Errorf("API.OK should set given status code\nexpected: %v\ngot: %v\n", expectedCode, response.Code)
	}

	// [Test] responseBody
	gotBody, err := ioutil.ReadAll(response.Body)

	if err != nil {
		panic(err)
	}

	gotBodyString := strings.TrimSuffix(string(gotBody), "\n")

	if gotBodyString != expectedBodyString {
		t.Errorf("API.OK should marshal given resource and write it\nexpected: %v\ngot: %v\n", expectedBodyString, gotBodyString)
	}
}

func TestAPIOkSet200IfGiven0AsHTTPStatusCode(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a := newTestAPI()

	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.Ok(w, "", 0)
	})

	a.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("API.OK should set 200 if given 0\nexpected: %v\ngot: %v\n", http.StatusOK, response.Code)
	}

}

func TestAPIErrorSetGivenHTTPStatusCodeAndResponseBody(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	expectedBodyString := "stubed testAPIMarshal"
	expectedCode := http.StatusNotFound

	// stub testAPI.Marshal
	testAPIMarshal = func(v interface{}) ([]byte, error) {
		return []byte(expectedBodyString), nil
	}

	a := newTestAPI()

	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.Error(w, "", expectedCode)
	})

	a.ServeHTTP(response, request)

	// [Test] http status code
	if response.Code != expectedCode {
		t.Errorf("API.Error should set given status code\nexpected: %v\ngot: %v\n", expectedCode, response.Code)
	}

	// [Test] responseBody
	gotBody, err := ioutil.ReadAll(response.Body)

	if err != nil {
		panic(err)
	}

	gotBodyString := strings.TrimSuffix(string(gotBody), "\n")

	if gotBodyString != expectedBodyString {
		t.Errorf("API.Error should marshal given resource and write it\nexpected: \"%v\"\ngot: \"%v\"\n", expectedBodyString, gotBodyString)
	}
}

func TestAPIErrorSet500IfGiven0AsHTTPStatusCode(t *testing.T) {
	request, _ := http.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()

	a := newTestAPI()

	a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.Error(w, "", 0)
	})

	a.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Errorf("API.OK should set 200 if given 0\nexpected: %v\ngot: %v\n", http.StatusOK, response.Code)
	}

}

func newTestAPI() *API {
	Register("testapi", &testAPI{})

	defer Deregister("testapi")

	a, err := NewAPI("testapi")

	if err != nil {
		panic(err)
	}

	return a
}
