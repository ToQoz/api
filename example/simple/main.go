package main

import (
	"errors"
	"github.com/ToQoz/dou"
	_ "github.com/ToQoz/dou/jsonapi"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
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

func main() {
	defer teardown()

	api, err := dou.NewAPI("jsonapi")
	if err != nil {
		log.Fatal(err)
	}

	api.ReadTimeout = 10 * time.Second
	api.WriteTimeout = 10 * time.Second
	api.MaxHeaderBytes = 1 << 20

	// --- Map routes ---
	api.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			api.Ok(w, map[string]string{"hello": "world"}, http.StatusOK)
		case "/error":
			err := errors.New("some error occur")
			api.Error(w, newAPIError(err), http.StatusInternalServerError)
		case "/errors":
			var errs []error
			errs = append(errs, errors.New("1 error occur"))
			errs = append(errs, errors.New("2 error occur"))
			api.Error(w, newAPIErrors(errs), http.StatusInternalServerError)
		default:
			api.Error(w, map[string]string{"message": http.StatusText(http.StatusNotFound)}, http.StatusNotFound)
		}
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
