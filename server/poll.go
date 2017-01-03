package server

import (
	"strings"
	"time"

	"github.com/jcelliott/lumber"
)

// StartPolling polls clients at preconfigured interval.
// Examples:
//  StartPolling(nil, nil, 60, nil)
//  StartPolling(nil, []string{"cpu"}, 1, ch)
//  StartPolling([]string{"computer1", "computer2"}, []string{"cpu"}, 1, ch)
func StartPolling(ids, tags []string, interval time.Duration, done chan struct{}) {
	lumber.Trace("[PULSE :: SERVER] StartPolling...")
	tick := time.Tick(interval)

	// getstat allows us to poll without waiting for the tick
	// since we can't send to receive only `tick` channel.
	getstat := func() {
		if ids == nil {
			if tags == nil {
				PollAll()
				return
			}

			Poll(tags)
			return
		}

		// todo: what is this for even?
		// newIds := []string{}
		// for _, sid := range findIds(tags) {
		// 	for _, id := range ids {
		// 		if id == sid {
		// 			newIds = append(newIds, id)
		// 		}
		// 	}
		// }
		// lumber.Trace("==NEWIDS=============='%+v'", newIds)

		if tags != nil {
			command := "get " + strings.Join(tags, ",") + "\n"
			sendAll(command, ids)
		}
	}

	// fetch stat immediately (dont wait `interval`)
	getstat()

	for {
		select {
		case <-tick:
			getstat()
		case <-done:
			return
		}
	}
}

// Poll polls based on tags
func Poll(tags []string) {
	lumber.Trace("[PULSE :: SERVER] Poll...")
	if tags == nil {
		PollAll()
		return
	}
	ids := findIds(tags)
	lumber.Trace("[PULSE :: SERVER] ids - '%+q'", ids)
	if len(ids) > 0 {
		command := "get " + strings.Join(tags, ",") + "\n"
		sendAll(command, ids)
	}
	lumber.Trace("[PULSE :: SERVER] END Poll")
}

// PollAll polls all clients for registered collectors(stats to be collected)
func PollAll() {
	lumber.Trace("[PULSE :: SERVER] PollAll...")
	// todo: RLock()
	for _, client := range clients {
		command := "get " + strings.Join(client.collectorList(), ",") + "\n"
		if command == "get \n" {
			continue
		}
		go client.conn.Write([]byte(command))
	}
}
