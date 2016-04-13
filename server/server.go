package server

import (
	"bufio"
	"errors"
	"net"
	"strings"
	"fmt"

	"github.com/jcelliott/lumber"

	"github.com/nanopack/pulse/plexer"
)

var (
	MissingPublisher = errors.New("A publisher is needed")
	publish          Publisher
)

type (
	Publisher func(plexer.MessageSet) error
)

func Listen(address string, publisher Publisher) error {
	if publisher == nil {
		return MissingPublisher
	}

	publish = publisher

	serverSocket, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go func() {
		defer serverSocket.Close()
		// Continually listen for any incoming connections.
		for {
			conn, err := serverSocket.Accept()
			if err != nil {
				// if the connection stops working we should
				// panic so we never are in a state where we thing
				// its accepting and it isnt
				panic(err)
			}

			// handle each connection individually (non-blocking)
			go handleConnection(conn)
		}
	}()
	return nil
}
func handleConnection(conn net.Conn) {
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
	clients[id] = &client{conn: conn}
	conn.Write([]byte("ok\n"))

	// now handle commands and data
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSuffix(line, "\n")
		if line == "close" {
			return
		}
		split := strings.SplitN(line, " ", 2)
		if len(split) != 2 {
			continue
		}

		cmd := split[0]
		switch cmd {
		case "ok":
			lumber.Trace("[PULSE :: SERVER] OK: %v", split)
			// just an ack
		case "got":
			lumber.Trace("[PULSE :: SERVER] GOT: %v", split)
			stats := strings.Split(split[1], ",")

			metric := plexer.MessageSet{
				Tags:     []string{"metrics", "host:" + id},
				Messages: make([]plexer.Message, 0),
			}

			for _, stat := range stats {
				fmt.Println("stat", stat)
				splitStat := strings.Split(stat, ":")
				if len(splitStat) != 2 {
					// i can only handle key value
					continue
				}

				name := splitStat[0]
				splitName := strings.Split(name, "-")
				if len(splitName) != 2 {
					// the name didnt come in as collector-name
					continue
				}
				tags := clients[id].tagList(splitName[0])
				message := plexer.Message{
					ID:   splitName[1],
					Tags: tags,
					Data: splitStat[1],
				}

				metric.Messages = append(metric.Messages, message)
			}
			publish(metric)
		case "add":
			lumber.Trace("[PULSE :: SERVER] ADD: %v", split)
			if !strings.Contains(split[1], ":") {
				clients[id].add(split[1], []string{})
				continue
			}
			split = strings.SplitN(split[1], ":", 2)
			tags := strings.Split(split[1], ",")
			if split[1] == "" {
				tags = []string{}
			}
			clients[id].add(split[0], tags)

		case "remove":
			lumber.Trace("[PULSE :: SERVER] REMOVE: %v", split)
			clients[id].remove(split[1])
			// record that the remote does not have a stat available
		case "close":
			lumber.Trace("[PULSE :: SERVER] CLOSE: %v", split)
			// clean shutoff of the connection
			delete(clients, id)
		default:
			lumber.Trace("[PULSE :: SERVER] BAD: %v", split)
			conn.Write([]byte("unknown command\n"))
		}

	}
}

// returns the server ids associated with the collector name given
func findIds(collectors []string) []string {
	ids := make([]string, 0)
	for id, client := range clients {
		for _, collector := range collectors {
			if client.includes(collector) {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids
}

func sendAll(command string, ids []string) {
	for _, id := range ids {
		client, ok := clients[id]
		if ok {
			go client.conn.Write([]byte(command))
		}
	}
}
