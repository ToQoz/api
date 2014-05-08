// Copyright 2014 Takatoshi Matsumoto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package dou is tiny and flexible toolkit for creating a api server.

Simple usage. If you want to see more example, check github.com/ToQoz/dou/example/full

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

	func newApiError(err error) *apiError {
		return &apiError{Message: err.Error()}
	}

	type apiErrors struct {
		Errors []*apiError `json:"errors"`
	}

	func newApiErrors(errs []error) *apiErrors {
		aErrs := &apiErrors{}

		for _, err := range errs {
			aErrs.Errors = append(aErrs.Errors, newApiError(err))
		}

		return aErrs
	}

	func main() {
		defer teardown()

		api, err := dou.NewApi("jsonapi")
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
				api.Error(w, newApiError(err), http.StatusInternalServerError)
			case "/errors":
				var errs []error
				errs = append(errs, errors.New("1 error occur"))
				errs = append(errs, errors.New("2 error occur"))
				api.Error(w, newApiErrors(errs), http.StatusInternalServerError)
			default:
				api.Error(w, map[string]string{"message": http.StatusText(http.StatusNotFound)}, http.StatusNotFound)
			}
		})

		// --- Create listener ---
		// You can use utility, for example github.com/lestrrat/go-server-starter-listener etc.
		l, err := net.Listen("tcp", ":8099")

		if err != nil {
			log.Fatalf("Could not listen: %s", ":8099")
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

You can creating a custom plugin in accordance with your api type or domain-specific use-case.
The plugin should keep following interface.

	type Plugin interface {
		OnPanic(w http.ResponseWriter, r *http.Request)
		BeforeDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request)
		AfterDispatch(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request)
		Marshal(v interface{}) ([]byte, error)
		Unmarshal(data []byte, v interface{}) error
		ApiStatus(w http.ResponseWriter, code int)
	}
*/
package dou
