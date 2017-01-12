// Package server handles the tcp socket api for interacting with clients(relays).
// It also handles polling clients based on registered tags.
package server

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jcelliott/lumber"
	"github.com/spf13/viper"

	"github.com/nanopack/pulse/plexer"
)

var (
	MissingPublisher = errors.New("A publisher is needed")
	publish          Publisher
)

type (
	Publisher func(plexer.MessageSet) error
)

// Listen starts the pulse tcp socket api (stats)
func Listen(address string, publisher Publisher) error {
	if publisher == nil {
		return MissingPublisher
	}

	publish = publisher

	serverSocket, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	lumber.Info("[PULSE :: SERVER] Listening at %s...", address)

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

func readData(conn net.Conn, dataChan chan string, errChan chan error) {
	zero := time.Time{}
	for {
		// make a temporary bytes var to read from the connection
		tmp := make([]byte, 128)
		// make 0 length data bytes (since we'll be appending)
		data := make([]byte, 0)

		// loop through the connection stream, appending tmp to data
		for {
			// readDeadline 2x as long as heartbeat
			conn.SetReadDeadline(time.Now().Add(time.Duration(viper.GetInt("beat-interval")*2) * time.Second))

			// read to the tmp var
			n, err := conn.Read(tmp)
			if err != nil {
				errChan <- err
				return
			}

			// append read data to full data
			data = append(data, tmp[:n]...)

			// break if ends with '\n' (todo: need to ensure writing w/o "\n" works)
			if tmp[n-1] == '\n' { //|| tmp[n-1] == 'EOF' {
				break
			}
		}

		conn.SetReadDeadline(zero)

		// return strings.TrimSuffix(string(data), "\n"), nil
		datas := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		for i := range datas {
			dataChan <- datas[i]
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	dataChan := make(chan string)
	errChan := make(chan error)

	go readData(conn, dataChan, errChan)

	var line string

	// read id
	select {
	case line = <-dataChan:
	case err := <-errChan:
		lumber.Debug("Failed to read from connection - %s", err)
		return
	}

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
	defer delete(clients, id)

	clients[id] = &client{conn: conn}
	conn.Write([]byte("ok\n"))

	// update client with configured beat-interval
	conn.Write([]byte(fmt.Sprintf("beat %d\n", viper.GetInt("beat-interval"))))

	// now handle commands and data
	for {
		select {
		case line = <-dataChan:
			if line == "close" {
				lumber.Trace("[PULSE :: SERVER] CLOSE: %s", line)
				return
			}

			if line == "ping" {
				lumber.Trace("[PULSE :: SERVER] PING: %s", line)
				conn.Write([]byte("pong\n"))
				continue
			}

			split := strings.SplitN(line, " ", 2)
			if len(split) != 2 {
				lumber.Trace("[PULSE :: SERVER] Not enough data: %v", split)
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
					// stat may be "test-test:25.25"
					splitStat := strings.Split(stat, ":")
					if len(splitStat) != 2 {
						// i can only handle key value
						continue
					}
					// splitstat would be ["test-test", "25.25"]

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
				return
			default:
				lumber.Trace("[PULSE :: SERVER] BAD: %v", split)
				// don't spam network
			}
		case err := <-errChan:
			lumber.Trace("[PULSE :: SERVER] ERROR: %s", err)
			return
		}
	}
}

// returns the server ids associated with the collector name given
func findIds(collectors []string) []string {
	ids := make([]string, 0)
	// todo: RLock()
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
	lumber.Trace("[PULSE :: SERVER] sendAll...")
	for _, id := range ids {
		client, ok := clients[id]
		if ok {
			go func() {
				_, err := client.conn.Write([]byte(command))
				if err != nil {
					lumber.Trace("[PULSE :: SERVER] sendAll: Error - %s", err)
					delete(clients, id)
					client.conn.Close()
				}
			}()
		}
	}
}
