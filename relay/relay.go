// Package provides the client the ability to connect to pulse and add
// metrics/stats to be collected.
package relay

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

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

// establishConnection establishes a connection and id's with the server
func (relay *Relay) establishConnection() (*bufio.Reader, error) {
	conn, err := net.Dial("tcp", relay.hostAddr)
	if err != nil {
		return nil, err
	}

	// send id
	conn.Write([]byte(fmt.Sprintf("id %v\n", relay.myId)))

	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "ok\n" {
		return nil, UnableToIdentify
	}

	// on successful id, hand over good connection to relay
	relay.conn = conn

	// add relay's known collectors
	for name, value := range relay.collectors {
		relay.conn.Write([]byte(fmt.Sprintf("add %s:%s\n", name, strings.Join(value.tags, ","))))
	}

	return r, nil
}

// NewRelay creates a new relay
func NewRelay(address, id string) (*Relay, error) {
	relay := &Relay{
		connected:  true,
		collectors: make(map[string]taggedCollector, 0),
		hostAddr:   address,
		myId:       id,
	}
	r, err := relay.establishConnection()
	if err != nil {
		return nil, err
	}

	go relay.runLoop(r)

	return relay, nil
}

// runLoop handles communication from the server
func (relay *Relay) runLoop(reader *bufio.Reader) {
	for {
		// when implementing relay, set `lumber.Level(lumber.LvlInt("TRACE"))` in client to view logs
		line, err := reader.ReadString('\n')
		if err != nil {
			lumber.Error("[PULSE :: RELAY] Disconnected from host %v!", relay.hostAddr)
			// retry indefinitely
			for {
				if reader, err = relay.establishConnection(); err == nil {
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
			lumber.Trace("[PULSE :: RELAY] GET: %v", split)
			if len(split) != 2 {
				continue
			}
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
			response := fmt.Sprintf("got %s\n", strings.Join(results, ","))
			relay.conn.Write([]byte(response))
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
	// todo: test drawbacks of removing '-' from check
	if name == "_connected" || strings.ContainsAny(name, ":,") {
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
	relay.collectors[name] = taggedCollector{collector: collector, tags: tags}
	return nil
}

func (relay *Relay) RemoveCollector(name string) {
	_, found := relay.collectors[name]
	if found {
		delete(relay.collectors, name)
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
