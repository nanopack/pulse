package main

import (
	"github.com/jcelliott/lumber"
	"github.com/nanopack/mist/core"
	"github.com/nanopack/pulse/api"
	"github.com/nanopack/pulse/plexer"
	"github.com/nanopack/pulse/server"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var configFile string

func main() {
	viper.SetDefault("server_listen_address", "127.0.0.1:3000")
	viper.SetDefault("token", "secret")
	viper.SetDefault("http_listen_address", "127.0.0.1:8080")
	viper.SetDefault("mist_address", "127.0.0.1:1234")
	viper.SetDefault("influx_address", "127.0.0.1:8086")
	viper.SetDefault("log_level", "INFO")
	viper.SetDefault("poll_interval", 60)
	// 

	server := true
	configFile := ""
	command := cobra.Command{
		Use:   "lojack",
		Short: "lojack is a server controller",
		Long:  ``,
		Run: func(ccmd *cobra.Command, args []string) {
			if !server {
				ccmd.HelpFunc()(ccmd, args)
				return
			}
			viper.SetConfigFile(configFile)
			viper.ReadInConfig()
			Viper.Set("log", lumber.NewConsoleLogger(lumber.LvlInt(viper.GetString("log_level"))))
			serverStart()
		},
	}
	command.Flags().BoolVarP(&server, "server", "s", false, "Run as server")
	command.Flags().StringVarP(&configFile, "configFile", "", "","config file location for server")

	command.Execute()
}

func serverStart() {
	
	plex := plexer.NewPlexer()

	if viper.GetString("mist_address") != "" {
		mist, err := mist.NewRemoteClient(viper.GetString("mist_address"))
		if err != nil {
			panic(err)
		}
		plex.AddObserver("mist", mist.Publish)
		defer mist.Close()
	}

	plex.AddBatcher("influx", server.InfluxInsert)

	server, err := server.Listen(viper.GetString("server_listen_address"), plex.Publish)
	if err != nil {
		panic(err)
	}
	defer server.Close()

	queries := []string{
		"CREATE DATABASE statistics",
		`CREATE RETENTION POLICY "2.days" ON statistics DURATION 2d REPLICATION 1 DEFAULT`,
		`CREATE RETENTION POLICY "1.week" ON statistics DURATION 1w REPLICATION 1`,
		// `CREATE CONTINUOUS QUERY "15minute_compile" ON statistics BEGIN select mean(cpu_used) as cpu_used, mean(ram_used) as ram_used, mean(swap_used) as swap_used, mean(disk_used) as disk_used, mean(disk_io_read) as disk_io_read, mean(disk_io_write) as disk_io_write, mean(disk_io_busy) as disk_io_busy, mean(disk_io_wait) as disk_io_wait into "1.week"."metrics" from "2.days"."metrics" group by time(15m), service END`,
	}

	for _, query := range queries {
		resp, err := server.Query(query)
		if err != nil {
			panic(err)
		}
	}
	
	err = api.Start()
	if err != nil {
		panic(err)
	}
}
