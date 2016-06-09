// Copyright (C) Pagoda Box, Inc - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential

// Package api provides a restful interface to view aggregated stats as well as manage alerts.
package api

//
import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/pat"
	"github.com/jcelliott/lumber"
	"github.com/nanobox-io/golang-nanoauth"
	"github.com/spf13/viper"
)

// structs
type (
	apiError struct {
		ErrorString string `json:"error"`
	}
	apiMsg struct {
		MsgString string `json:"msg"`
	}
)

var (
	BadJson      = errors.New("Bad JSON syntax received in body")
	BodyReadFail = errors.New("Body Read Failed")
)

// start sets the state of the package if the config has all the necessary data for the api
// and starts the default api server; routing web requests and handling all the routes
func Start() error {
	routes, err := registerRoutes()
	if err != nil {
		return err
	}

	// blocking...
	if viper.GetBool("insecure") {
		lumber.Info("[PULSE :: API] Listening at 'http://%s'...\n", viper.GetString("http-listen-address"))
		return http.ListenAndServe(viper.GetString("http-listen-address"), routes)
	}
	lumber.Info("[PULSE :: API] Listening at 'https://%s'...\n", viper.GetString("http-listen-address"))
	nanoauth.DefaultAuth.Header = "X-AUTH-TOKEN"
	return nanoauth.ListenAndServeTLS(viper.GetString("http-listen-address"), viper.GetString("token"), routes)
}

// registerRoutes
func registerRoutes() (*pat.Router, error) {
	lumber.Debug("[PULSE :: API] Registering routes...")

	//
	router := pat.New()

	//
	router.Get("/ping", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("pong"))
	})

	router.Get("/keys", keysRequest)
	router.Get("/tags", tagsRequest)

	router.Get("/latest/{stat}", latestStat)
	router.Get("/hourly/{stat}", hourlyStat)
	router.Get("/daily/{stat}", dailyStat)
	router.Get("/daily_peaks/{stat}", dailyStat)

	// only expose alert routes if alerting configured
	if viper.GetString("kapacitor-address") != "" {
		// todo: maybe get and list tasks from kapacitor
		router.Post("/alerts", setAlert)
		router.Put("/alerts", setAlert)
		router.Delete("/alerts/{id}", deleteAlert)
	}

	return router, nil
}

// writeBody
func writeBody(v interface{}, rw http.ResponseWriter, status int, req *http.Request) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// print the error only if there is one
	var msg map[string]string
	json.Unmarshal(b, &msg)

	var errMsg string
	if msg["error"] != "" {
		errMsg = msg["error"]
	}

	lumber.Debug(`[PULSE :: ACCESS] %v - [%v] %v %v %v - "User-Agent: %s" %s`,
		req.RemoteAddr, req.Proto, req.Method, req.RequestURI,
		status, req.Header.Get("User-Agent"), errMsg)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	rw.Write(append(b, byte('\n')))

	return nil
}

// parseBody parses the json body into v
func parseBody(req *http.Request, v interface{}) error {

	// read the body
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		lumber.Error(err.Error())
		return BodyReadFail
	}
	defer req.Body.Close()

	// parse body and store in v
	err = json.Unmarshal(b, v)
	if err != nil {
		lumber.Error(err.Error())
		return BadJson
	}

	return nil
}
