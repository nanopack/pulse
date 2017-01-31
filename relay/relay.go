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
	beatInterval       = 30
)

type (
	// Relay is a pulse client
	Relay struct {
		conn       net.Conn
		dataChan   chan string
		errChan    chan error
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

func (relay *Relay) readData() {
	zero := time.Time{}
	for {
		// make a temporary bytes var to read from the connection
		tmp := make([]byte, 128)
		// make 0 length data bytes (since we'll be appending)
		data := make([]byte, 0)

		// loop through the connection stream, appending tmp to data
		for {
			// read to the tmp var
			n, err := relay.conn.Read(tmp)
			if err != nil {
				relay.errChan <- err
				return
			}

			// append read data to full data
			data = append(data, tmp[:n]...)

			// break if ends with '\n' (todo: need to ensure writing w/o "\n" works)
			if tmp[n-1] == '\n' { //|| tmp[n-1] == 'EOF' {
				break
			}
		}

		relay.conn.SetReadDeadline(zero)

		// return strings.TrimSuffix(string(data), "\n"), nil
		datas := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		for i := range datas {
			relay.dataChan <- datas[i]
		}
	}
}

// beat allows the detection and handling of stale tcp connections
func (relay *Relay) beat() {
	for {
		time.Sleep(time.Duration(beatInterval) * time.Second)
		// since we're always reading, lets set a timeout for the pong to come back in 1/2 beat time
		relay.conn.SetReadDeadline(time.Now().Add(time.Duration(beatInterval/2) * time.Second))
		lumber.Trace("[PULSE :: RELAY] PULSE pinging...")
		_, err := relay.conn.Write([]byte("ping\n"))
		if err != nil {
			lumber.Trace("[PULSE :: RELAY] PULSE ping failed - %s", err)
			relay.errChan <- err
			return
		}
		lumber.Trace("[PULSE :: RELAY] PULSE pinged!")
	}
}

// establishConnection establishes a connection and id's with the server
func (relay *Relay) establishConnection() error {
	conn, err := net.DialTimeout("tcp", relay.hostAddr, 10*time.Second)
	if err != nil {
		return err
	}

	// send id
	conn.Write([]byte(fmt.Sprintf("id %s\n", relay.myId)))

	// hand over connection to client (relay)
	relay.conn = conn

	relay.dataChan = make(chan string)
	relay.errChan = make(chan error)

	// start data reader
	go relay.readData()

	// start heartbeat
	go relay.beat()

	var line string

	select {
	case line = <-relay.dataChan:
	case err := <-relay.errChan:
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

// NewRelay creates a new client (relay)
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
		select {
		case err := <-relay.errChan:
			lumber.Error("[PULSE :: RELAY] Disconnected from host %s!", relay.hostAddr)
			relay.conn.Close()
			// retry indefinitely
			for {
				if err = relay.establishConnection(); err == nil {
					lumber.Info("[PULSE :: RELAY] Reconnected to host %s!", relay.hostAddr)
					break
				}
				lumber.Debug("[PULSE :: RELAY] Reconnecting to host %s...  Fail!", relay.hostAddr)
				<-time.After(5 * time.Second)
			}
			// we won't have anything in 'line' so continue
			continue
		case line := <-relay.dataChan:
			line = strings.TrimSuffix(line, "\n")
			split := strings.SplitN(line, " ", 2)

			cmd := split[0]
			switch cmd {
			case "ok":
				lumber.Trace("[PULSE :: RELAY] OK: %s", split)
				// just an ack
			case "pong":
				lumber.Trace("[PULSE :: RELAY] PONG: %s", split)
			case "beat":
				lumber.Trace("[PULSE :: RELAY] BEAT: %s", split)
				if len(split) != 2 {
					continue
				}
				num, err := strconv.Atoi(split[1])
				if err == nil {
					beatInterval = num
				}
			case "get":
				if len(split) != 2 {
					continue
				}
				lumber.Trace("[PULSE :: RELAY] GET: %s", split)
				stats := strings.Split(split[1], ",")
				results := make([]string, 0)
				for _, stat := range stats {
					tagCollector, ok := relay.collectors[stat]
					if !ok {
						lumber.Trace("[PULSE :: RELAY] stat %s !ok", stat)
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
					_, err := relay.conn.Write([]byte(response))
					if err != nil {
						lumber.Trace("[PULSE :: RELAY] GET response write error - %s", err)
					}
				}
			default:
				lumber.Trace("[PULSE :: RELAY] BAD: %s", split)
				// causes network spam if we write anything to connection
			}
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
		lumber.Trace("[PULSE :: RELAY] Failed to add collector to server - %s", err)
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
		if _, err := relay.conn.Write([]byte(fmt.Sprintf("remove %s\n", name))); err != nil {
			lumber.Trace("[PULSE :: RELAY] Failed to remove collector from server - %s", err)
		}
	}
}

func (relay *Relay) Close() error {
	for name := range relay.collectors {
		relay.RemoveCollector(name)
	}
	relay.conn.Write([]byte("close\n"))
	return relay.conn.Close()
}
