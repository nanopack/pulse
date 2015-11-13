package relay

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/nanopack/pulse/collector"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	UnableToIdentify = errors.New("unable to identify with pulse")
	ReservedName     = errors.New("name is reserved")
)

type (
	Relay struct {
		conn       net.Conn
		collectors map[string]collector.Collector
		connected  bool
	}
)

func NewRelay(address, id string) (*Relay, error) {
	// should this reconnect?
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
		collectors: make(map[string]collector.Collector, 0),
		connected:  true,
	}

	go relay.runLoop(r)

	return relay, nil
}

func (relay *Relay) runLoop(reader *bufio.Reader) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSuffix(line, "\n")
		split := strings.SplitN(line, " ", 2)

		cmd := split[0]
		switch cmd {
		case "ok":
			// just an ack
		case "get":
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
				for name, value := range collector.Values() {
					formatted := strconv.FormatFloat(value, 'f', 4, 64)
					switch {
					case name == "":
						results = append(results, stat+":"+formatted)
					default:
						results = append(results, stat+"-"+name+":"+formatted)
					}

				}

			}
			response := strings.Join(results, ",")
			relay.conn.Write(append([]byte("got "), append([]byte(response), '\n')...))
		case "flush":
			for _, collector := range relay.collectors {
				collector.Flush()
			}
			relay.conn.Write([]byte("ok\n"))
		case "override":
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
				collector.OverrideInterval(time.Duration(value), duration)
			}
			relay.conn.Write([]byte("ok\n"))
		default:
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
		values := stat.Values()
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

func (relay *Relay) AddCollector(name string, collector collector.Collector) error {
	switch {
	case name == "_connected":
		return ReservedName
	}
	relay.RemoveCollector(name)
	relay.collectors[name] = collector
	collector.Start()
	relay.conn.Write([]byte(fmt.Sprintf("add %v\n", name)))
	return nil
}

func (relay *Relay) RemoveCollector(name string) {
	collector, found := relay.collectors[name]
	if found {
		collector.Stop()
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
