package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ToQoz/dou"
	_ "github.com/ToQoz/dou/jsonapi"
	"github.com/ToQoz/rome"
	"github.com/lestrrat/go-apache-logformat"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	APIStatusOk              = 1
	APIStatusValidationError = 100
	APIStatusUnexpectedError = 900
	logger                   = apachelog.CombinedLog
)

// --- API Error type ---

type apiError struct {
	Message string `json:"message"`
}

func newAPIError(err error) *apiError {
	return &apiError{Message: err.Error()}
}

type apiErrors struct {
	Errors []*apiError `json:"errors"`
}

func newAPIErrors(errs []error) *apiErrors {
	aErrs := &apiErrors{}

	for _, err := range errs {
		aErrs.Errors = append(aErrs.Errors, newAPIError(err))
	}

	return aErrs
}

// --- Example struct ---

var users = []*User{}

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (u *User) Validate() []error {
	var errs []error

	if u.Name == "" {
		errs = append(errs, errors.New("user: name is required"))
	}

	if u.Email == "" {
		errs = append(errs, errors.New("user: email is required"))
	}

	return errs
}

func (u *User) Save() error {
	users = append(users, u)
	return nil
}

func main() {
	defer teardown()

	// --- Setup Router ---
	// ! You can use router keeping interface `api.Router` instead of github.com/ToQoz/rome
	router := rome.NewRouter()
	router.NotFoundFunc(func(w http.ResponseWriter, r *http.Request) {
		j, err := json.Marshal(map[string]string{
			"message":           http.StatusText(http.StatusNotFound),
			"documentation_url": "http://toqoz.net",
		})

		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		// skip wrote bytesyze
		_, err = fmt.Fprintln(w, string(j))

		if err != nil {
			log.Printf("dou: fail to fmt.Fpintln(http.ResponseWriter, string)\n%v", err)
		}
	})

	// --- Setup API ---
	api, err := dou.NewAPI("jsonapi")
	api.Handler = router
	//api, err := dou.NewAPI("jsonapi")
	if err != nil {
		log.Fatal(err)
	}

	api.BeforeDispatch = func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
		// Call default
		w, r = api.Plugin.BeforeDispatch(w, r)

		lw := apachelog.NewLoggingWriter(w, r, logger)
		return lw, r
	}

	api.AfterDispatch = func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
		// Call default
		w, r = api.Plugin.AfterDispatch(w, r)

		if lw, ok := w.(*apachelog.LoggingWriter); ok {
			lw.EmitLog()
		}

		return w, r
	}

	api.ReadTimeout = 10 * time.Second
	api.WriteTimeout = 10 * time.Second
	api.MaxHeaderBytes = 1 << 20

	// --- Map routes ---
	router.GetFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		api.APIStatus(w, APIStatusOk)
		api.Ok(w, users, http.StatusOK)
	})

	router.GetFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		api.APIStatus(w, APIStatusUnexpectedError)
		api.Error(w, map[string]string{"message": "Internal server error"}, http.StatusInternalServerError)
	})

	// Try Ok    $ curl -X POST -d 'name=ToQoz&email=toqoz403@gmail.com' -D - :8099/users
	// Try Error $ curl -X POST -D - :8099/users
	router.PostFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		u := &User{
			Name:  r.FormValue("name"),
			Email: r.FormValue("email"),
		}

		errs := u.Validate()

		if len(errs) > 0 {
			api.APIStatus(w, APIStatusValidationError)
			api.Error(w, newAPIErrors(errs), 422)
			return
		}

		err := u.Save()

		if err != nil {
			api.APIStatus(w, APIStatusUnexpectedError)
			api.Error(w, newAPIErrors(errs), http.StatusInternalServerError)
			return
		}

		api.APIStatus(w, APIStatusOk)
		api.Ok(w, u, http.StatusCreated)
	})

	// --- Create listener ---
	// You can use utility, for example github.com/lestrrat/go-server-starter-listener etc.
	l, err := net.Listen("tcp", ":8099")

	if err != nil {
		log.Printf("Could not listen: %s", ":8099")
		teardown()
		os.Exit(1)
	}

	log.Printf("Listen: %s", ":8099")

	// --- Handle C-c ---
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for sig := range c {
			log.Print("Stopping the server...")

			switch sig {
			case os.Interrupt:
				// --- Stop Server ---
				api.Stop()
				return
			default:
				log.Print("Receive unknown signal...")
			}
		}
	}()

	// --- Run Server ---
	api.Run(l)
}

func teardown() {
	log.Print("Tearing down...")
	log.Print("Finished - bye bye.  ;-)")
}
