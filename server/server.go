package server

import (
	"bufio"
	"errors"

	"net"
	"strings"

	"github.com/nanopack/pulse/plexer"
)

var (
	MissingPublisher = errors.New("A publisher is needed")
	publish Publisher
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
			publish(metric)
		case "add":
			// record that the remote has a stat available
			clients[id].add(split[1])
		case "remove":
			clients[id].remove(split[1])
			// record that the remote does not have a stat available
		case "close":
			// clean shutoff of the connection
			delete(clients, id)
		default:
			conn.Write([]byte("unknown command\n"))
		}

	}
}

// returns the server ids associated with the tags given
func findIds(tags []string) []string {
	ids := make([]string, 0)
	for id, client := range clients {
		for _, tag := range tags {
			if client.includes(tag) {
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
