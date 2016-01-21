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

	"github.com/nanopack/pulse/collector"
)

var (
	UnableToIdentify   = errors.New("unable to identify with pulse")
	ReservedName       = errors.New("cannot use - or : or , or _connected in your name")
	DuplicateCollector = errors.New("cannot add a duplicate collector to the set")
)

type (
	Relay struct {
		conn       net.Conn
		collectors map[string]valuer
		connected  bool
		hostAddr   string
		myId       string
	}

	valuer struct {
		id   collector.Collector
		tags []string
	}
)

func NewRelay(address, id string) (*Relay, error) {

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	conn.Write([]byte(fmt.Sprintf("id %v\n", id)))

	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')

	if err != nil {
		return nil, err
	}
	if line != "ok\n" {
		return nil, UnableToIdentify
	}

	relay := &Relay{
		conn:       conn,
		connected:  true,
		collectors: make(map[string]valuer, 0),
		hostAddr:   address,
		myId:       id,
	}

	go relay.runLoop(r)

	return relay, nil
}

func (relay *Relay) runLoop(reader *bufio.Reader) {
	for {
		// when implementing relay, set `lumber.Level(lumber.LvlInt("TRACE"))` in client to view logs
		line, err := reader.ReadString('\n')
		if err != nil {
			lumber.Trace("[PULSE :: RELAY] Disconnected from host %v!", relay.hostAddr)
			// maybe keep looping infinitely, since program doesn't exit
			for i := 0; i < 20; i++ {
				if newRelay, err := NewRelay(relay.hostAddr, relay.myId); err == nil {
					lumber.Trace("[PULSE :: RELAY] Reconnected to host %v!", relay.hostAddr)
					relay = newRelay
					return
				}
				lumber.Trace("[PULSE :: RELAY] Reconnecting to host %v...  Fail!", relay.hostAddr)
				<-time.After(6 * time.Second)
			}
			lumber.Error("[PULSE :: RELAY] Failed to reconnect to host %v! Giving up.", relay.hostAddr)
			// do this?
			relay.Close()
			return
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
				collector, ok := relay.collectors[stat]
				if !ok {
					continue
				}
				for name, value := range collector.id.Values() {
					formatted := strconv.FormatFloat(value, 'f', 4, 64)
					switch {
					case name == "":
						results = append(results, stat+":"+formatted)
					default:
						results = append(results, stat+"-"+name+":"+formatted)
					}

				}

			}
			response := fmt.Sprintf("got %s\n", strings.Join(results, ","))
			relay.conn.Write([]byte(response))
		case "flush":
			lumber.Trace("[PULSE :: RELAY] FLUSH: %v", split)
			for _, collector := range relay.collectors {
				collector.id.Flush()
			}
			relay.conn.Write([]byte("ok\n"))
		case "override":
			lumber.Trace("[PULSE :: RELAY] OVERRIDE: %v", split)
			args := strings.SplitN(split[1], " ", 2)
			params := strings.Split(args[1], ",")
			value, err := strconv.Atoi(args[0])
			if err != nil {
				relay.conn.Write([]byte("bad argument\n"))
			}
			duration := time.Second * time.Duration(value)
			for _, param := range params {
				stat := strings.Split(param, ":")
				collector, ok := relay.collectors[stat[0]]
				if !ok {
					continue
				}
				value, err = strconv.Atoi(stat[1])
				if err != nil {
					relay.conn.Write([]byte("bad argument\n"))
				}
				collector.id.OverrideInterval(time.Duration(value), duration)
			}
			relay.conn.Write([]byte("ok\n"))
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
		values := stat.id.Values()
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

func (relay *Relay) AddCollector(name string, tags []string, collector collector.Collector) error {
	if name == "_connected" || strings.ContainsAny(name, "-:,") {
		return ReservedName
	}
	if _, ok := relay.collectors[name]; ok {
		return DuplicateCollector
	}
	relay.collectors[name] = valuer{id: collector, tags: tags}
	collector.Start()
	relay.conn.Write([]byte(fmt.Sprintf("add %s:%s\n", name, strings.Join(tags, ","))))
	return nil
}

func (relay *Relay) RemoveCollector(name string) {
	collector, found := relay.collectors[name]
	if found {
		delete(relay.collectors, name)
		collector.id.Stop()
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
