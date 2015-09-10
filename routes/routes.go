// -*- mode: go; tab-width: 2; indent-tabs-mode: 1; st-rulers: [70] -*-
// vim: ts=4 sw=4 ft=lua noet
//--------------------------------------------------------------------
// @author Daniel Barney <daniel@nanobox.io>
// Copyright (C) Pagoda Box, Inc - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly
// prohibited. Proprietary and confidential
//
// @doc
//
// @end
// Created :   31 August 2015 by Daniel Barney <daniel@nanobox.io>
//--------------------------------------------------------------------
package routes

import (
	"bitbucket.org/nanobox/na-api"
	"bitbucket.org/nanobox/na-pulse/server"
	"encoding/json"
	"fmt"
	"net/http"
)

func Init() {
	api.Router.Get("/services/{service}/stats/{stat}/hourly", api.TraceRequest(statRequest))
	api.Router.Get("/services/{service}/stats/{stat}/daily_peaks", api.TraceRequest(combinedRequest))
}

func statRequest(res http.ResponseWriter, req *http.Request) {
	server := api.User.(*server.Server)
	service := req.URL.Query().Get(":service")
	stat := req.URL.Query().Get(":stat")
	query := fmt.Sprintf(`select "%v" from "1.week".metrics where service = '%v'`, stat, service)
	fmt.Println(query)
	records, err := server.Query(query)
	if err != nil {

		return
	}

	bytes, err := json.Marshal(<-records)
	if err != nil {

	}
	res.Write(bytes)
}

func combinedRequest(res http.ResponseWriter, req *http.Request) {

}
