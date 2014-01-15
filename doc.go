// Copyright 2014 Takatoshi Matsumoto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package github.com/ToQoz/api is json api tools

Router used by api should be keep following interface

	type Router interface {
		Get(string, http.Handler)
		Head(string, http.Handler)
		Post(string, http.Handler)
		Put(string, http.Handler)
		Delete(string, http.Handler)
		http.Handler
	}

Usage. (use github.com/ToQoz/rome as Router)

	package main

	import (
		"encoding/json"
		"github.com/ToQoz/api"
		_ "github.com/ToQoz/api/jsonapi"
		"github.com/ToQoz/rome"
		"log"
		"net"
		"net/http"
		"os"
		"os/signal"
		"time"
	)

	var (
		ApiUnexpectedError = 100
	)

	func main() {
		// --- Setup API ---
		api, err := api.NewApi("jsonapi", rome.NewRouter())
		if err != nil {
			log.Fatal(err)
		}

		api.ReadTimeout = 10 * time.Second
		api.WriteTimeout = 10 * time.Second
		api.MaxHeaderBytes = 1 << 20

		// --- GET / ---
		api.Get("/", func(w http.ResponseWriter, r *http.Request) {
			j, err := json.Marshal(map[string]string{"hello":"world"})

			if err != nil {
				api.Error(w, err, http.StatusInternalServerError, ApiUnexpectedError)
				return
			}

			w.Write(j)
		})

		// --- Create listener ---
		// You can use utility, for example github.com/lestrrat/go-server-starter-listener etc.
		addr := ":8099"
		l, err := net.Listen("tcp", addr)

		if err != nil {
			log.Fatalf("Could not listen: %s", addr)
		}

		// --- Handle C-c ---
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		go func() {
			for sig := range c {
				log.Print("Stopping the server...")

				switch sig {
				case os.Interrupt:
					log.Print("Tearing down...")

					// --- Stop Server ---
					api.Stop()

					log.Fatal("Finished - bye bye.  ;-)")
				default:
					log.Fatal("Receive unknown signal...")
				}
			}
		}()

		// --- Run Server ---
		api.Run(l)
	}
*/
package api
