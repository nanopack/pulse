package main

import (
	"fmt"
	"time"

	"github.com/jcelliott/lumber"
	mist "github.com/nanopack/mist/clients"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nanopack/pulse/api"
	"github.com/nanopack/pulse/influx"
	"github.com/nanopack/pulse/kapacitor"
	"github.com/nanopack/pulse/plexer"
	pulse "github.com/nanopack/pulse/server"
)

var (
	// for clarity
	httpAddress       = "127.0.0.1:8080"
	serverAddress     = "127.0.0.1:3000"
	influxAddress     = "http://127.0.0.1:8086"
	kapacitorAddress  = "" // "http://127.0.0.1:9092"
	mistAddress       = ""
	mistToken         = ""
	logLevel          = "INFO"
	server            = false
	insecure          = true
	token             = "secret"
	pollInterval      = 60
	aggregateInterval = 15

	configFile = ""
	version    = false

	// Pulse is the pulse cli
	Pulse = &cobra.Command{
		Use:   "pulse",
		Short: "pulse is a stat collecting and publishing service",
		Long:  ``,

		PersistentPreRunE: readConfig,
		PreRunE:           preFlight,
		RunE:              startPulse,
		SilenceErrors:     true,
		SilenceUsage:      true,
	}

	// to be populated by go linker
	tag    string
	commit string
)

func init() {
	Pulse.Flags().StringP("http-listen-address", "H", httpAddress, "Http listen address")
	viper.BindPFlag("http-listen-address", Pulse.Flags().Lookup("http-listen-address"))
	Pulse.Flags().StringP("server-listen-address", "S", serverAddress, "Server listen address")
	viper.BindPFlag("server-listen-address", Pulse.Flags().Lookup("server-listen-address"))
	Pulse.Flags().StringP("influx-address", "i", influxAddress, "InfluxDB server address")
	viper.BindPFlag("influx-address", Pulse.Flags().Lookup("influx-address"))
	Pulse.Flags().StringP("kapacitor-address", "k", kapacitorAddress, "Kapacitor server address (http://127.0.0.1:9092)")
	viper.BindPFlag("kapacitor-address", Pulse.Flags().Lookup("kapacitor-address"))
	Pulse.Flags().StringP("mist-address", "m", mistAddress, "Mist server address")
	viper.BindPFlag("mist-address", Pulse.Flags().Lookup("mist-address"))
	Pulse.Flags().StringP("mist-token", "M", mistToken, "Mist server address")
	viper.BindPFlag("mist-token", Pulse.Flags().Lookup("mist-token"))
	Pulse.Flags().StringP("log-level", "l", logLevel, "Level at which to log")
	viper.BindPFlag("log-level", Pulse.Flags().Lookup("log-level"))
	Pulse.Flags().BoolP("server", "s", server, "Run as server")
	viper.BindPFlag("server", Pulse.Flags().Lookup("server"))
	Pulse.Flags().BoolP("insecure", "I", insecure, "Run insecure")
	viper.BindPFlag("insecure", Pulse.Flags().Lookup("insecure"))

	Pulse.Flags().StringP("token", "t", token, "Security token (recommend placing in config file)")
	viper.BindPFlag("token", Pulse.Flags().Lookup("token"))
	Pulse.Flags().IntP("poll-interval", "p", pollInterval, "Interval to request stats from clients")
	viper.BindPFlag("poll-interval", Pulse.Flags().Lookup("poll-interval"))
	Pulse.Flags().IntP("aggregate-interval", "a", aggregateInterval, "Interval at which stats are aggregated")
	viper.BindPFlag("aggregate-interval", Pulse.Flags().Lookup("aggregate-interval"))

	Pulse.Flags().StringVarP(&configFile, "configFile", "c", configFile, "Config file location for server")
	Pulse.Flags().BoolVarP(&version, "version", "v", version, "Print version info and exit")

	lumber.Level(lumber.LvlInt(viper.GetString("log-level")))
}

func preFlight(ccmd *cobra.Command, args []string) error {
	if version {
		fmt.Printf("pulse %s (%s)\n", tag, commit)
		return fmt.Errorf("") // no error, just exit
	}

	if !viper.GetBool("server") {
		ccmd.HelpFunc()(ccmd, args)
		return fmt.Errorf("") // no error, just exit
	}

	return nil
}

func readConfig(ccmd *cobra.Command, args []string) error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
		err := viper.ReadInConfig()
		if err != nil {
			lumber.Fatal("Failed to read config - %v", err)
			return err
		}
	}

	return nil
}

func main() {
	Pulse.Execute()
}

func startPulse(ccmd *cobra.Command, args []string) error {
	// re-initialize logger
	lumber.Level(lumber.LvlInt(viper.GetString("log-level")))

	plex := plexer.NewPlexer()

	if viper.GetString("mist-address") != "" {
		mist, err := mist.New(viper.GetString("mist-address"), viper.GetString("mist-token"))
		if err != nil {
			lumber.Fatal("Mist failed to start - %v", err.Error())
			return err
		}
		plex.AddObserver("mist", mist.Publish)
		defer mist.Close()
	}

	plex.AddBatcher("influx", influx.Insert)

	err := pulse.Listen(viper.GetString("server-listen-address"), plex.Publish)
	if err != nil {
		lumber.Fatal("Pulse failed to start - %v", err.Error())
		return err
	}
	// begin polling the connected servers
	pi := viper.GetInt("poll-interval")
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
			return err
		}
	}

	go influx.KeepContinuousQueriesUpToDate()

	if viper.GetString("kapacitor-address") != "" {
		err := kapacitor.Init()
		if err != nil {
			lumber.Fatal("Kapacitor failed to start - %v", err.Error())
			return err
		}
	}

	err = api.Start()
	if err != nil {
		lumber.Fatal("Api failed to start - %v", err.Error())
		return err
	}

	return nil
}
