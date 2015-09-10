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
package main

import (
	"bitbucket.org/nanobox/na-api"
	"bitbucket.org/nanobox/na-pulse/plexer"
	"bitbucket.org/nanobox/na-pulse/poller"
	"bitbucket.org/nanobox/na-pulse/routes"
	"bitbucket.org/nanobox/na-pulse/server"
	"fmt"
	"github.com/jcelliott/lumber"
	"github.com/pagodabox/golang-mist"
	"github.com/pagodabox/nanobox-config"
	"os"
	"strings"
	"time"
)

func main() {
	configFile := ""
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		configFile = os.Args[1]
	}

	defaults := map[string]string{
		"server_listen_addres": "127.0.0.1:1234",
		"http_listen_address":  "127.0.0.1:8080",
		"mist_address":         "127.0.0.1:1234",
		"log_level":            "INFO",
	}

	config.Load(defaults, configFile)
	config := config.Config

	level := lumber.LvlInt(config["log_level"])

	api.Name = "PULSE"
	api.Logger = lumber.NewConsoleLogger(level)
	api.User = nil

	mist, err := mist.NewRemoteClient(config["mist_address"])
	if err != nil {
		panic(err)
	}
	defer mist.Close()

	plex := plexer.NewPlexer()

	plex.AddObserver("mist", mist.Publish)

	server, err := server.Listen(config["server_listen_address"], plex.Publish)
	if err != nil {
		panic(err)
	}
	defer server.Close()

	influx, err := server.StartInfluxd()
	if err != nil {
		panic(err)
	}
	defer influx.Close()

	poller := poller.NewPoller(server.Poll)
	client := poller.NewClient()
	defer client.Close()

	client.Poll("cpu", 60)
	client.Poll("ram", 60)
	client.Poll("disk", 60)

	routes.Init()

	time.Sleep(time.Second * 2)

	resChan, err := server.Query("CREATE DATABASE statistics")
	if err != nil {
		panic(err)
	}
	fmt.Println(<-resChan)

	resChan, err = server.Query("CREATE RETENTION POLICY yearSingle ON statistics DURATION 365d REPLICATION 1 DEFAULT")
	if err != nil {
		panic(err)
	}
	fmt.Println(<-resChan)

	plex.AddBatcher("influx", server.InfluxInsert)

	api.Start(config["http_listen_address"])
}
