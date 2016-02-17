package main

import (
	"os"
	"time"

	"github.com/jcelliott/lumber"
	"github.com/nanopack/mist/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nanopack/pulse/api"
	"github.com/nanopack/pulse/influx"
	"github.com/nanopack/pulse/plexer"
	"github.com/nanopack/pulse/server"
)

var configFile string

func main() {
	viper.SetDefault("server_listen_address", "127.0.0.1:3000")
	viper.SetDefault("http_listen_address", "127.0.0.1:8080")
	viper.SetDefault("mist_address", "")
	viper.SetDefault("influx_address", "http://127.0.0.1:8086")
	viper.SetDefault("log_level", "INFO")
	// config file only
	viper.SetDefault("token", "secret")
	viper.SetDefault("poll_interval", 60)
	viper.SetDefault("aggregate_interval", 15)

	server := true
	configFile := ""
	command := cobra.Command{
		Use:   "pulse",
		Short: "pulse is a stat collecting and publishing service",
		Long:  ``,
		Run: func(ccmd *cobra.Command, args []string) {
			if !server {
				ccmd.HelpFunc()(ccmd, args)
				return
			}
			viper.SetConfigFile(configFile)
			viper.ReadInConfig()
			lumber.Level(lumber.LvlInt(viper.GetString("log_level")))

			serverStart()
		},
	}
	command.Flags().String("http_listen_address", ":8080", "Http listen address")
	viper.BindPFlag("http_listen_address", command.Flags().Lookup("http_listen_address"))
	command.Flags().String("influx_address", "127.0.0.1:8086", "InfluxDB server address")
	viper.BindPFlag("influx_address", command.Flags().Lookup("influx_address"))
	command.Flags().String("mist_address", "", "Mist server address")
	viper.BindPFlag("mist_address", command.Flags().Lookup("mist_address"))
	command.Flags().String("log_level", "INFO", "Level at which to log")
	viper.BindPFlag("log_level", command.Flags().Lookup("log_level"))
	command.Flags().BoolVarP(&server, "server", "s", false, "Run as server")
	command.Flags().StringVarP(&configFile, "configFile", "", "", "Config file location for server")

	command.Execute()
}

func serverStart() {

	plex := plexer.NewPlexer()

	if viper.GetString("mist_address") != "" {
		mist, err := mist.NewRemoteClient(viper.GetString("mist_address"))
		if err != nil {
			lumber.Fatal(err.Error())
			os.Exit(1)
		}
		plex.AddObserver("mist", mist.Publish)
		defer mist.Close()
	}

	plex.AddBatcher("influx", influx.Insert)

	err := server.Listen(viper.GetString("server_listen_address"), plex.Publish)
	if err != nil {
		lumber.Fatal(err.Error())
		os.Exit(1)
	}
	// begin polling the connected servers
	pi := viper.GetInt("poll_interval")
	if pi == 0 {
		pi = 60
	}
	go server.StartPolling(nil, nil, time.Duration(pi)*time.Second, nil)

	queries := []string{
		"CREATE DATABASE statistics",
		`CREATE RETENTION POLICY "2.days" ON statistics DURATION 2d REPLICATION 1 DEFAULT`,
		`CREATE RETENTION POLICY "1.week" ON statistics DURATION 1w REPLICATION 1`,
	}

	for _, query := range queries {
		_, err := influx.Query(query)
		if err != nil {
			lumber.Fatal(err.Error())
			os.Exit(1)
		}
	}

	go influx.KeepContinuousQueriesUpToDate()

	err = api.Start()
	if err != nil {
		lumber.Fatal(err.Error())
		os.Exit(1)
	}
}
