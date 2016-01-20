package main

import (
	"time"
	
	"github.com/jcelliott/lumber"
	"github.com/nanopack/mist/core"
	"github.com/nanopack/pulse/api"
	"github.com/nanopack/pulse/plexer"
	"github.com/nanopack/pulse/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nanopack/pulse/influx"
)

var configFile string

func main() {
	viper.SetDefault("server_listen_address", "127.0.0.1:3000")
	viper.SetDefault("token", "secret")
	viper.SetDefault("http_listen_address", "127.0.0.1:8080")
	viper.SetDefault("mist_address", "")
	viper.SetDefault("influx_address", "http://127.0.0.1:8086")
	viper.SetDefault("log_level", "INFO")
	viper.SetDefault("poll_interval", 60)
	viper.SetDefault("aggregate_interval", 15)

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
			viper.Set("log", lumber.NewConsoleLogger(lumber.LvlInt(viper.GetString("log_level"))))
			serverStart()
		},
	}
	command.Flags().String("http_listen_address", ":8080", "Http Listen address")
	viper.BindPFlag("http_listen_address", command.Flags().Lookup("http_listen_address"))
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

	plex.AddBatcher("influx", influx.Insert)

	err := server.Listen(viper.GetString("server_listen_address"), plex.Publish)
	if err != nil {
		panic(err)
	}
	// begin polling the connected servers
	pi := viper.GetInt("poll_interval")
	if pi == 0 {
		pi = 60
	}
	go server.StartPolling(nil, nil, time.Duration(pi) * time.Second, nil)


	queries := []string{
		"CREATE DATABASE statistics",
		`CREATE RETENTION POLICY "2.days" ON statistics DURATION 2d REPLICATION 1 DEFAULT`,
		`CREATE RETENTION POLICY "1.week" ON statistics DURATION 1w REPLICATION 1`,
	}

	for _, query := range queries {
		_, err := influx.Query(query)
		if err != nil {
			panic(err)
		}
	}

	err = api.Start()
	if err != nil {
		panic(err)
	}

	go influx.KeepContinuousQueriesUpToDate()
}
