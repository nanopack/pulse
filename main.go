// Pulse is a stat collecting and publishing service. It serves historical
// stats over an http api while live stats are sent to mist for live updates.
//
// Usage
//
// To start pulse as a server, simply run (with influx running locally):
//
//  pulse -s
//
// For more specific usage information, refer to the help doc `pulse -h`:
//  Usage:
//    pulse [flags]
//
//  Flags:
//    -a, --aggregate-interval int         Interval at which stats are aggregated (default 15)
//    -c, --config-file string              Config file location for server
//    -C, --cors-allow string              Sets the 'Access-Control-Allow-Origin' header (default "*")
//    -H, --http-listen-address string     Http listen address (default "127.0.0.1:8080")
//    -i, --influx-address string          InfluxDB server address (default "http://127.0.0.1:8086")
//    -I, --insecure                       Run insecure (default true)
//    -k, --kapacitor-address string       Kapacitor server address (http://127.0.0.1:9092)
//    -l, --log-level string               Level at which to log (default "INFO")
//    -m, --mist-address string            Mist server address
//    -M, --mist-token string              Mist server token
//    -p, --poll-interval int              Interval to request stats from clients (default 60)
//    -s, --server                         Run as server
//    -S, --server-listen-address string   Server listen address (default "127.0.0.1:3000")
//    -t, --token string                   Security token (recommend placing in config file) (default "secret")
//    -v, --version                        Print version info and exit
//
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
	logLevel          = "info"
	corsAllow         = "*"
	server            = false
	insecure          = true
	token             = "secret"
	pollInterval      = 60
	beatInterval      = 30 // heartbeat frequency (seconds)
	aggregateInterval = 15
	retention         = 1

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
	Pulse.Flags().StringP("mist-token", "M", mistToken, "Mist server token")
	viper.BindPFlag("mist-token", Pulse.Flags().Lookup("mist-token"))
	Pulse.Flags().StringP("log-level", "l", logLevel, "Level at which to log")
	viper.BindPFlag("log-level", Pulse.Flags().Lookup("log-level"))
	Pulse.Flags().StringP("cors-allow", "C", corsAllow, "Sets the 'Access-Control-Allow-Origin' header")
	viper.BindPFlag("cors-allow", Pulse.Flags().Lookup("cors-allow"))
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
	Pulse.Flags().IntP("retention", "r", retention, "Number of weeks to store aggregated stats")
	viper.BindPFlag("retention", Pulse.Flags().Lookup("retention"))
	Pulse.Flags().IntP("beat-interval", "b", beatInterval, "Heartbeat frequency (seconds)")
	viper.BindPFlag("beat-interval", Pulse.Flags().Lookup("beat-interval"))

	Pulse.Flags().StringVarP(&configFile, "config-file", "c", configFile, "Config file location for server")
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
			return fmt.Errorf("Failed to read config - %s", err)
		}
	}

	// validate retention
	if viper.GetInt("retention") < 1 {
		fmt.Println("Bad value for retention, resetting to 1")
		viper.Set("retention", 1)
	}

	// validate beat-interval
	if viper.GetInt("beat-interval") < 1 {
		fmt.Println("Bad value for beat-interval, resetting to 30")
		viper.Set("retention", 30)
	}

	return nil
}

func main() {
	// print errors
	err := Pulse.Execute()
	if err != nil && err.Error() != "" {
		fmt.Println(err)
	}
}

func startPulse(ccmd *cobra.Command, args []string) error {
	// re-initialize logger
	lumber.Level(lumber.LvlInt(viper.GetString("log-level")))

	plex := plexer.NewPlexer()

	if viper.GetString("mist-address") != "" {
		mist, err := mist.New(viper.GetString("mist-address"), viper.GetString("mist-token"))
		if err != nil {
			return fmt.Errorf("Mist failed to start - %s", err)
		}
		plex.AddObserver("mist", mist.Publish)
		defer mist.Close()
	}

	plex.AddBatcher("influx", influx.Insert)

	err := pulse.Listen(viper.GetString("server-listen-address"), plex.Publish)
	if err != nil {
		return fmt.Errorf("Pulse failed to start - %s", err)
	}
	// begin polling the connected servers
	pollSec := viper.GetInt("poll-interval")
	if pollSec == 0 {
		pollSec = 60
	}
	go pulse.StartPolling(nil, nil, time.Duration(pollSec)*time.Second, nil)

	queries := []string{
		"CREATE DATABASE statistics",
		"CREATE RETENTION POLICY one_day ON statistics DURATION 8h REPLICATION 1 DEFAULT",
		fmt.Sprintf("CREATE RETENTION POLICY one_week ON statistics DURATION %dw REPLICATION 1", viper.GetInt("retention")), // todo: ALTER as well?
	}

	for _, query := range queries {
		_, err := influx.Query(query)
		if err != nil {
			return fmt.Errorf("Failed to query influx - %s", err)
		}
	}

	go influx.KeepContinuousQueriesUpToDate()

	if viper.GetString("kapacitor-address") != "" {
		err := kapacitor.Init()
		if err != nil {
			return fmt.Errorf("Kapacitor failed to start - %s", err)
		}
	}

	err = api.Start()
	if err != nil {
		return fmt.Errorf("Api failed to start - %s", err)
	}

	return nil
}
