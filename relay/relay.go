// Package provides the client the ability to connect to pulse and add
// metrics/stats to be collected.
package relay

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	// when implementing relay, set `lumber.Level(lumber.LvlInt("TRACE"))` in client to view logs
	"github.com/jcelliott/lumber"
)

var (
	UnableToIdentify   = errors.New("unable to identify with pulse")
	ReservedName       = errors.New("cannot use - or : or , or _connected in your name")
	DuplicateCollector = errors.New("cannot add a duplicate collector to the set")
)

type (
	// Relay is a pulse client
	Relay struct {
		conn       net.Conn
		collectors map[string]taggedCollector
		connected  bool
		hostAddr   string
		myId       string
	}

	// stores the collector and its associated tags
	taggedCollector struct {
		collector Collector
		tags      []string
	}
)

func (relay *Relay) readData() (string, error) {
	// make a temporary bytes var to read from the connection
	tmp := make([]byte, 128)
	// make 0 length data bytes (since we'll be appending)
	data := make([]byte, 0)

	// loop through the connection stream, appending tmp to data
	for {
		// read to the tmp var
		n, err := relay.conn.Read(tmp)
		if err != nil {
			return "", err
		}

		// append read data to full data
		data = append(data, tmp[:n]...)

		// break if ends with '\n' (todo: not as sure as delimited reading)
		if tmp[n-1] == '\n' {
			break
		}
	}

	return strings.TrimSuffix(string(data), "\n"), nil
}

// establishConnection establishes a connection and id's with the server
func (relay *Relay) establishConnection() error {
	conn, err := net.Dial("tcp", relay.hostAddr)
	if err != nil {
		return err
	}

	// send id
	conn.Write([]byte(fmt.Sprintf("id %s\n", relay.myId)))

	// hand over connection to client (relay)
	relay.conn = conn

	line, err := relay.readData()
	if err != nil {
		return err
	}

	if line != "ok" {
		return UnableToIdentify
	}

	// add relay's known collectors
	for name, value := range relay.collectors {
		relay.conn.Write([]byte(fmt.Sprintf("add %s:%s\n", name, strings.Join(value.tags, ","))))
	}

	return nil
}

// NewRelay creates a new relay
func NewRelay(address, id string) (*Relay, error) {
	newRelay := &Relay{
		connected:  true,
		collectors: make(map[string]taggedCollector, 0),
		hostAddr:   address,
		myId:       id,
	}
	err := newRelay.establishConnection()
	if err != nil {
		return nil, err
	}

	go newRelay.runLoop()

	return newRelay, nil
}

// runLoop handles communication from the server
func (relay *Relay) runLoop() {
	for {
		line, err := relay.readData()
		if err != nil {
			lumber.Error("[PULSE :: RELAY] Disconnected from host %v!", relay.hostAddr)
			// retry indefinitely
			for {
				if err = relay.establishConnection(); err == nil {
					lumber.Info("[PULSE :: RELAY] Reconnected to host %v!", relay.hostAddr)
					break
				}
				lumber.Debug("[PULSE :: RELAY] Reconnecting to host %v...  Fail!", relay.hostAddr)
				<-time.After(5 * time.Second)
			}
			// we won't have anything in 'line' so continue
			continue
		}

		line = strings.TrimSuffix(line, "\n")
		split := strings.SplitN(line, " ", 2)

		cmd := split[0]
		switch cmd {
		case "ok":
			lumber.Trace("[PULSE :: RELAY] OK: %v", split)
			// just an ack
		case "get":
			if len(split) != 2 {
				continue
			}
			lumber.Trace("[PULSE :: RELAY] GET: %v", split)
			stats := strings.Split(split[1], ",")
			results := make([]string, 0)
			for _, stat := range stats {
				tagCollector, ok := relay.collectors[stat]
				if !ok {
					continue
				}
				for name, value := range tagCollector.collector.Collect() {
					formatted := strconv.FormatFloat(value, 'f', 4, 64)
					if name == "" {
						name = stat
					}
					results = append(results, fmt.Sprintf("%s-%s:%s", stat, name, formatted))
				}
			}
			if len(results) > 0 {
				response := fmt.Sprintf("got %s\n", strings.Join(results, ","))
				relay.conn.Write([]byte(response))
			}
		default:
			lumber.Trace("[PULSE :: RELAY] BAD: %v", split)
			relay.conn.Write([]byte("unknown command\n"))
		}
	}
}

func (relay *Relay) Info() map[string]float64 {
	stats := make(map[string]float64, 2)
	stats["_connected"] = 0
	if relay.connected {
		stats["_connected"] = 1
	}
	for collection, stat := range relay.collectors {
		values := stat.collector.Collect()
		for name, value := range values {
			switch {
			case name == "":
				stats[collection] = value
			default:
				stats[collection+"-"+name] = value
			}
		}
	}
	return stats
}

// AddCollector adds a collector to relay
func (relay *Relay) AddCollector(name string, tags []string, collector Collector) error {
	if name == "_connected" || strings.ContainsAny(name, "-:,") {
		lumber.Trace("[PULSE :: RELAY] Reserved name!")
		return ReservedName
	}
	if _, ok := relay.collectors[name]; ok {
		lumber.Trace("[PULSE :: RELAY] Duplicate collector!")
		return DuplicateCollector
	}
	if _, err := relay.conn.Write([]byte(fmt.Sprintf("add %s:%s\n", name, strings.Join(tags, ",")))); err != nil {
		lumber.Trace("[PULSE :: RELAY] Failed to write!")
		return err
	}

	// if successfully added collector, add it to relay's known collectors
	// todo: lock
	relay.collectors[name] = taggedCollector{collector: collector, tags: tags}
	lumber.Trace("[PULSE :: RELAY] Added '%s' as collector.", name)
	return nil
}

func (relay *Relay) RemoveCollector(name string) {
	_, found := relay.collectors[name]
	if found {
		// todo: lock
		delete(relay.collectors, name)
		lumber.Trace("[PULSE :: RELAY] Removed '%s' as collector.", name)
		relay.conn.Write([]byte(fmt.Sprintf("remove %v\n", name)))
	}
}

func (relay *Relay) Close() error {
	for name := range relay.collectors {
		relay.RemoveCollector(name)
	}
	relay.conn.Write([]byte("close\n"))
	return relay.conn.Close()
}
