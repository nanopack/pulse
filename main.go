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
	"bitbucket.org/nanobox/nanobox-config"
	"github.com/jcelliott/lumber"
	"github.com/pagodabox/golang-mist"
	"github.com/shirou/gopsutil"
	"os"
	"strings"
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

	poller := poller.NewPoller(server.Poll)
	client := poller.NewClient()
	defer client.Close()

	client.Poll("cpu", 60)
	client.Poll("ram", 60)
	client.Poll("disk", 60)

	routes.Init()
	api.Start(config["http_listen_address"])
}
