package server

import (
	"bufio"
	"errors"

	"io"
	"net"
	"strings"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"

	"github.com/nanopack/pulse/plexer"
)

// to be replaced with config.Influx
var InfluxServer = "http://localhost:8086"

var (
	MissingPublisher = errors.New("A publisher is needed")
)

type (
	Publisher func(plexer.MessageSet) error
	Server    struct {
		influxClient influx.Client
		// I need a map that stores which client has which data points available
		publish     Publisher
		conn        io.Closer
		mappings    map[string]map[string]interface{}
		connections map[string]net.Conn
	}
)

func Listen(address string, publisher Publisher) (*Server, error) {
	if publisher == nil {
		return nil, MissingPublisher
	}
	if address == "" {
		address = "127.0.0.1:1445"
	}
	serverSocket, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	server := &Server{
		publish:     publisher,
		conn:        serverSocket,
		mappings:    make(map[string]map[string]interface{}, 0),
		connections: make(map[string]net.Conn),
	}

	// use client instead of server in our server
	server.influxClient, err = influx.NewHTTPClient(influx.HTTPConfig{
		Addr: "http://localhost:8086",
	})
	if err != nil {
		panic(err)
	}
	defer server.influxClient.Close()

	go func() {
		defer serverSocket.Close()
		// Continually listen for any incoming connections.
		for {
			conn, err := serverSocket.Accept()
			if err != nil {
				// what should we do with the error?
				return
			}

			// handle each connection individually (non-blocking)
			go handleConnection(server, conn)
		}
	}()
	return server, nil
}

func (server *Server) Close() error {
	return server.conn.Close()
}

func handleConnection(server *Server, conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return
	}
	line = strings.TrimSuffix(line, "\n")
	split := strings.SplitN(line, " ", 2)
	if split[0] != "id" {
		conn.Write([]byte("identify first with the 'id' command\n"))
		return
	}
	if len(split) != 2 {
		conn.Write([]byte("missing id\n"))
		return
	}

	id := split[1]
	server.mappings[id] = make(map[string]interface{})
	server.connections[id] = conn

	conn.Write([]byte("ok\n"))

	// now handle commands and data
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSuffix(line, "\n")
		split := strings.SplitN(line, " ", 2)
		if len(split) != 2 {
			continue
		}

		cmd := split[0]
		switch cmd {
		case "ok":
			// just an ack
		case "got":
			stats := strings.Split(split[1], ",")

			metric := plexer.MessageSet{
				Tags:     []string{"type:metrics", "service:" + id}, // TODO host tag may not be right
				Messages: make([]plexer.Message, 0),
			}

			for _, stat := range stats {
				splitStat := strings.Split(stat, ":")
				if len(splitStat) != 2 {
					return
				}

				message := plexer.Message{
					Tags: splitStat[:1],
					Data: splitStat[1],
				}

				metric.Messages = append(metric.Messages, message)
			}
			server.publish(metric)
		case "add":
			// record that the remote has a stat available
			server.mappings[id][split[1]] = true
		case "remove":
			delete(server.mappings[id], split[1])
			// record that the remote does not have a stat available
		case "close":
			// clean shutoff of the connection
			delete(server.mappings, id)
		default:
			conn.Write([]byte("unknown command\n"))
		}

	}
}

func (server *Server) Override(override map[string]int, duration time.Duration) {
	tags := make([]string, len(override))
	pairs := make([]string, len(override))
	for key, override := range override {
		tags = append(tags, key)
		pairs = append(pairs, key+":"+string(override))

	}
	command := "override " + string(duration) + " " + strings.Join(pairs, ",") + "\n"
	ids := server.findIds(tags)
	server.sendAll(command, ids)
}

func (server *Server) Poll(tags []string) {
	ids := server.findIds(tags)
	command := "get " + strings.Join(tags, ",") + "\n"
	server.sendAll(command, ids)
}

func (server *Server) findIds(tags []string) []string {
	ids := make([]string, 0)
	for id, checkTags := range server.mappings {
		for _, tag := range tags {
			if _, ok := checkTags[tag]; ok {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids
}

func (server *Server) sendAll(command string, ids []string) {
	for _, id := range ids {
		connection, ok := server.connections[id]
		if ok {
			go connection.Write([]byte(command))
		}
	}
}
