package server

import (
	"time"
	"strings"
)

// StartPolling(nil, nil, 60, ch)
// StartPolling(nil, []string{"cpu"}, 1, ch)
// StartPolling([]string{"computer1", "computer2"}, []string{"cpu"}, 1, ch)
func (server *Server) StartPolling(ids, tags []string, interval time.Duration, done chan struct{}) {
	tick := time.Tick(interval)
	for {
		select {
			case <-tick:
				if ids != nil {
					server.Poll(tags)
					continue
				}

				newIds := []string{}
				for _, sid := range server.findIds(tags) {
					for _, id := range ids {
						if id == sid {
							newIds = append(newIds, id)
						}
					}
				}
				command := "get " + strings.Join(tags, ",") + "\n"
				server.sendAll(command, ids)
			case <-done:
				return
		}
	}
}

func (server *Server) Poll(tags []string) {
	if tags == nil {
		server.PollAll()
		return
	}
	ids := server.findIds(tags)
	command := "get " + strings.Join(tags, ",") + "\n"
	server.sendAll(command, ids)
}

func (server *Server) PollAll() {
	for id, conn := range server.connections {
		tags := []string{}
		for tag, _ := range server.mappings[id] {
			tags = append(tags, tag)
		}
		command := "get " + strings.Join(tags, ",") + "\n"
		go conn.Write([]byte(command))
	}	
}
