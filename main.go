package main

import (
	"github.com/jcelliott/lumber"
	"github.com/nanobox-io/nanobox-api"
	"github.com/nanopack/mist/core"
	"github.com/nanopack/pulse/plexer"
	"github.com/nanopack/pulse/poller"
	"github.com/nanopack/pulse/routes"
	"github.com/nanopack/pulse/server"
	"github.com/pagodabox/nanobox-config"
	"os"
	"strings"
)

func main() {
	configFile := ""
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		configFile = os.Args[1]
	}

	defaults := map[string]string{
		"server_listen_address": "127.0.0.1:3000",
		"http_listen_address":   "127.0.0.1:8080",
		"mist_address":          "127.0.0.1:1234",
		"log_level":             "INFO",
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

	// store externally in redis
	polling_intervals := map[string]uint{
		"cpu_used":      60,
		"ram_used":      60,
		"swap_used":     60,
		"disk_used":     60,
		"disk_io_read":  60,
		"disk_io_write": 60,
		"disk_io_busy":  60,
		"disk_io_wait":  60,
	}

	for name, interval := range polling_intervals {
		client.Poll(name, interval)
	}

	api.Name = "PULSE"
	api.User = server
	routes.Init()

	queries := []string{
		"CREATE DATABASE statistics",
		`CREATE RETENTION POLICY "2.days" ON statistics DURATION 2d REPLICATION 1 DEFAULT`,
		`CREATE RETENTION POLICY "1.week" ON statistics DURATION 1w REPLICATION 1 DEFAULT`,
		`CREATE CONTINUOUS QUERY "15minute_compile" ON statistics BEGIN select mean(cpu_used) as cpu_used, mean(ram_used) as ram_used, mean(swap_used) as swap_used, mean(disk_used) as disk_used, mean(disk_io_read) as disk_io_read, mean(disk_io_write) as disk_io_write, mean(disk_io_busy) as disk_io_busy, mean(disk_io_wait) as disk_io_wait into "1.week"."metrics" from "2.days"."metrics" group by time(15m), service END`,
	}

	for _, query := range queries {
		// what to do with results?
		_, err := server.Query(query)
		if err != nil {
			panic(err)
		}
	}

	plex.AddBatcher("influx", server.InfluxInsert)

	api.Start(config["http_listen_address"])
}
