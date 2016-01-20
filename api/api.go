// Copyright (C) Pagoda Box, Inc - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
package api

//
import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/pat"
	"github.com/jcelliott/lumber"
	"github.com/nanobox-io/nanoauth"
	"github.com/spf13/viper"
)

// structs
type (
	//
	API struct {
	}
)

var defaultAPI = &API{}

// start sets the state of the package if the config has all the necessary data for the api
// and starts the default api server
func Start() error {
	return defaultAPI.Start()
}

// start routing web requests and handling all the routes
func (api *API) Start() error {
	routes, err := api.registerRoutes()
	if err != nil {
		return err
	}

	//
	lumber.Info("[PULSE :: API] Listening at %v\n", viper.GetString("http_listen_address"))

	// blocking...
	return nanoauth.ListenAndServeTLS(viper.GetString("http_listen_address"), viper.GetString("token"), routes)
}

// registerRoutes
func (api *API) registerRoutes() (*pat.Router, error) {
	lumber.Debug("[PULSE :: API] Registering routes...\n")

	//
	router := pat.New()

	//
	router.Get("/ping", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("pong"))
	})

	router.Get("/services/{service}/stats/{stat}/hourly", api.handleRequest(statRequest))
	router.Get("/services/{service}/stats/{stat}/daily_peaks", api.handleRequest(combinedRequest))

	return router, nil
}

// handleRequest
func (api *API) handleRequest(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {

		lumber.Trace(`
Request:
--------------------------------------------------------------------------------
%+v

`, req)

		//
		fn(rw, req)

		lumber.Trace(`
Response:
--------------------------------------------------------------------------------
%+v

`, rw)
	}
}

// writeBody
func writeBody(v interface{}, rw http.ResponseWriter, status int) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	rw.Write(b)

	return nil
}
