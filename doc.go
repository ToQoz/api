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
		"github.com/ToQoz/api"
		"github.com/ToQoz/rome"
		"os"
	)

	var (
		addr = func() (port string) {
			port = os.Getenv("PORT")

			if port == "" {
				port = ":8099"
			}
			return
		}()
		host = func() (host string) {
			host = os.Getenv("HOST")

			if host == "" {
				host = "localhost"
			}
			return
		}
	)

	func main() {
		api := api.NewApi(rome.NewRouter())

		api.Post("/users", func(w http.ResponseWriter, r *http.Request) {
			params := foil.NewWrappedParams(r)

			user, errs := CreateUser(
				params.Value("name"),
				params.Value("email"),
				params.Value("password"),
			)

			if errs != nil {
				api.Errors(w, errs)
				return
			}

			j, err := json.Marshal(user)

			if err != nil {
				api.Error(w, err)
				return
			}

			w.Header().Set("Location", fmt.Sprintf("http://%s%s/users/%d", host, addr, user.Id))
			w.WriteHeader(http.StatusCreated)
			w.Write(j)
		})

		api.Run(addr)
	}

	type User struct {}

	func CreateUser() (*User, error) {
		return &User{}, nil
	}
*/
package api
