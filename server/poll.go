package server

import (
	"time"
	"strings"
)

// StartPolling(nil, nil, 60, nil)
// StartPolling(nil, []string{"cpu"}, 1, ch)
// StartPolling([]string{"computer1", "computer2"}, []string{"cpu"}, 1, ch)
func StartPolling(ids, tags []string, interval time.Duration, done chan struct{}) {
	tick := time.Tick(interval)
	for {
		select {
			case <-tick:
				if ids != nil {
					Poll(tags)
					continue
				}

				newIds := []string{}
				for _, sid := range findIds(tags) {
					for _, id := range ids {
						if id == sid {
							newIds = append(newIds, id)
						}
					}
				}
				command := "get " + strings.Join(tags, ",") + "\n"
				sendAll(command, ids)
			case <-done:
				return
		}
	}
}

func Poll(tags []string) {
	if tags == nil {
		PollAll()
		return
	}
	ids := findIds(tags)
	command := "get " + strings.Join(tags, ",") + "\n"
	sendAll(command, ids)
}

func PollAll() {
	for _, client := range clients {
		command := "get " + strings.Join(client.tags, ",") + "\n"
		go client.conn.Write([]byte(command))
	}	
}
