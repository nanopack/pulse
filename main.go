package main

import (
	"fmt"
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
	mist_token         string
	server_address     string
	server             bool
	version            bool
	token              = "secret"
	poll_interval      = 60
	aggregate_interval = 15

	Pulse = &cobra.Command{
		Use:   "pulse",
		Short: "pulse is a stat collecting and publishing service",
		Long:  ``,

		Run: startPulse,
	}

	// to be populated by go linker
	tag string
	commit string
)

func init() {
	lumber.Level(lumber.LvlInt(viper.GetString("log_level")))
}

func startPulse(ccmd *cobra.Command, args []string) {
	if version {
		fmt.Printf("pulse %s (%s)", tag, commit)
		return
	}

	if configFile != "" {
		viper.SetConfigFile(configFile)
		err := viper.ReadInConfig()
		if err != nil {
			lumber.Fatal("Failed to read config - %v", err)
			return
			// os.Exit(1)
		}
	}

	if !server {
		ccmd.HelpFunc()(ccmd, args)
		return
	}

	// re-initialize logger
	lumber.Level(lumber.LvlInt(viper.GetString("log_level")))

	serverStart()
}

func main() {
	Pulse.Flags().StringVarP(&http_address, "http_listen_address", "H", "127.0.0.1:8080", "Http listen address")
	viper.BindPFlag("http_listen_address", Pulse.Flags().Lookup("http_listen_address"))
	Pulse.Flags().StringVarP(&server_address, "server_listen_address", "S", "127.0.0.1:3000", "Server listen address")
	viper.BindPFlag("server_listen_address", Pulse.Flags().Lookup("server_listen_address"))
	Pulse.Flags().StringVarP(&influx_address, "influx_address", "i", "http://127.0.0.1:8086", "InfluxDB server address")
	viper.BindPFlag("influx_address", Pulse.Flags().Lookup("influx_address"))
	Pulse.Flags().StringVarP(&mist_address, "mist_address", "m", "", "Mist server address")
	viper.BindPFlag("mist_address", Pulse.Flags().Lookup("mist_address"))
	Pulse.Flags().StringVarP(&mist_token, "mist_token", "M", "", "Mist server address")
	viper.BindPFlag("mist_token", Pulse.Flags().Lookup("mist_token"))
	Pulse.Flags().StringVarP(&log_level, "log_level", "l", "INFO", "Level at which to log")
	viper.BindPFlag("log_level", Pulse.Flags().Lookup("log_level"))
	Pulse.Flags().BoolVarP(&server, "server", "s", false, "Run as server")
	viper.BindPFlag("server", Pulse.Flags().Lookup("server"))

	Pulse.Flags().StringVarP(&token, "token", "t", "secret", "Security token (recommend placing in config file)")
	viper.BindPFlag("token", Pulse.Flags().Lookup("token"))
	Pulse.Flags().IntVarP(&poll_interval, "poll_interval", "p", 60, "Interval to request stats from clients")
	viper.BindPFlag("poll_interval", Pulse.Flags().Lookup("poll_interval"))
	Pulse.Flags().IntVarP(&aggregate_interval, "aggregate_interval", "a", 15, "Interval at which stats are aggregated")
	viper.BindPFlag("aggregate_interval", Pulse.Flags().Lookup("aggregate_interval"))

	Pulse.Flags().StringVarP(&configFile, "config_file", "c", "", "Config file location for server")
	Pulse.Flags().BoolVarP(&version, "version", "v", false, "Print version info and exit")

	Pulse.Execute()
}

func serverStart() {

	plex := plexer.NewPlexer()

	if viper.GetString("mist_address") != "" {
		mist, err := mist.New(viper.GetString("mist_address"), viper.GetString("mist_token"))
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
		`CREATE RETENTION POLICY two_days ON statistics DURATION 2d REPLICATION 1 DEFAULT`,
		`CREATE RETENTION POLICY one_week ON statistics DURATION 1w REPLICATION 1`,
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
