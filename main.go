package main

import (
	"os"
	"time"

	"github.com/jcelliott/lumber"
	mist "github.com/nanopack/mist/clients"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nanopack/pulse/api"
	"github.com/nanopack/pulse/influx"
	"github.com/nanopack/pulse/plexer"
	pulse "github.com/nanopack/pulse/server"
)

var (
	configFile         string
	http_address       string
	influx_address     string
	log_level          string
	mist_address       string
	server_address     string
	server             bool
	token              = "secret"
	poll_interval      = 60
	aggregate_interval = 15
)

func main() {

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

	command.Flags().StringVarP(&http_address, "http_listen_address", "H", "127.0.0.1:8080", "Http listen address")
	viper.BindPFlag("http_listen_address", command.Flags().Lookup("http_listen_address"))
	command.Flags().StringVarP(&server_address, "server_listen_address", "S", "127.0.0.1:3000", "Server listen address")
	viper.BindPFlag("server_listen_address", command.Flags().Lookup("server_listen_address"))
	command.Flags().StringVarP(&influx_address, "influx_address", "i", "http://127.0.0.1:8086", "InfluxDB server address")
	viper.BindPFlag("influx_address", command.Flags().Lookup("influx_address"))
	command.Flags().StringVarP(&mist_address, "mist_address", "m", "", "Mist server address")
	viper.BindPFlag("mist_address", command.Flags().Lookup("mist_address"))
	command.Flags().StringVarP(&log_level, "log_level", "l", "INFO", "Level at which to log")
	viper.BindPFlag("log_level", command.Flags().Lookup("log_level"))
	command.Flags().BoolVarP(&server, "server", "s", false, "Run as server")
	viper.BindPFlag("server", command.Flags().Lookup("server"))

	command.Flags().StringVarP(&token, "token", "t", "secret", "Security token (recommend placing in config file)")
	viper.BindPFlag("token", command.Flags().Lookup("token"))
	command.Flags().IntVarP(&poll_interval, "poll_interval", "p", 60, "Interval to request stats from clients")
	viper.BindPFlag("poll_interval", command.Flags().Lookup("poll_interval"))
	command.Flags().IntVarP(&aggregate_interval, "aggregate_interval", "a", 15, "Interval at which stats are aggregated")
	viper.BindPFlag("aggregate_interval", command.Flags().Lookup("aggregate_interval"))

	command.Flags().StringVarP(&configFile, "config_file", "c", "", "Config file location for server")


	command.Execute()
}

func serverStart() {

	plex := plexer.NewPlexer()

	if viper.GetString("mist_address") != "" {
		mist, err := mist.New(viper.GetString("mist_address"))
		if err != nil {
			lumber.Fatal("Mist failed to start - %v", err.Error())
			os.Exit(1)
		}
		plex.AddObserver("mist", mist.Publish)
		defer mist.Close()
	}

	plex.AddBatcher("influx", influx.Insert)

	err := pulse.Listen(viper.GetString("server_listen_address"), plex.Publish)
	if err != nil {
		lumber.Fatal("Pulse failed to start - %v", err.Error())
		os.Exit(1)
	}
	// begin polling the connected servers
	pi := viper.GetInt("poll_interval")
	if pi == 0 {
		pi = 60
	}
	go pulse.StartPolling(nil, nil, time.Duration(pi)*time.Second, nil)

	queries := []string{
		"CREATE DATABASE statistics",
		`CREATE RETENTION POLICY "2.days" ON statistics DURATION 2d REPLICATION 1 DEFAULT`,
		`CREATE RETENTION POLICY "1.week" ON statistics DURATION 1w REPLICATION 1`,
	}

	for _, query := range queries {
		_, err := influx.Query(query)
		if err != nil {
			lumber.Fatal("Failed to query influx - %v", err.Error())
			os.Exit(1)
		}
	}

	go influx.KeepContinuousQueriesUpToDate()

	err = api.Start()
	if err != nil {
		lumber.Fatal("Api failed to start - %v", err.Error())
		os.Exit(1)
	}
}
