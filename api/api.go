// Copyright (C) Pagoda Box, Inc - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
package api

//
import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/pat"
	"github.com/nanobox-io/nanoauth"
	"github.com/pborman/uuid"
)

// structs
type (
	//
	API struct {
	}
)

var defaultAPI = &API{}
var log lumber.Logger

// start sets the state of the package if the config has all the necessary data for the api
// and starts the default api server
func Start() error {
	return defaultAPI.Start()
}

// start routing web requests and handling all the routes
func (api *API) Start() error {
	log = viper.Get("log").(lumber.Logger)

	routes, err := api.registerRoutes()
	if err != nil {
		return err
	}

	//
	log.Info("[NANOBOX :: API] Listening on port %v\n", config.APIPort)

	// blocking...
	return nanoauth.ListenAndServeTLS(viper.GetString("http_listen_address"), viper.GetString("token"), routes)
}

// registerRoutes
func (api *API) registerRoutes() (*pat.Router, error) {
	log.Debug("[NANOBOX :: API] Registering routes...\n")

	//
	router := pat.New()

	//
	router.Get("/ping", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("pong"))
	})

	Get("/services/{service}/stats/{stat}/hourly", api.handleRequest(statRequest))
	Get("/services/{service}/stats/{stat}/daily_peaks", api.handleRequest(combinedRequest))	

	return router, nil
}

// handleRequest
func (api *API) handleRequest(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {

		log.Debug(`
Request:
--------------------------------------------------------------------------------
%+v

`, req)

		//
		fn(rw, req)

		log.Debug(`
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
