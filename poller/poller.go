package poller

import (
	"sync/atomic"
	"time"
)

type (
	Poll   func([]string)
	Client struct {
		id     uint32
		tags   map[string]uint
		poller *Poller
	}
	poll struct {
		after <-chan time.Time
		tags  []string
	}
	Poller struct {
		poll    Poll
		clients map[uint32]*Client
		done    chan bool
		next    uint32
	}
)

func NewPoller(poll Poll) *Poller {
	poller := &Poller{
		poll:    poll,
		clients: make(map[uint32]*Client, 0),
		done:    make(chan bool),
		next:    0,
	}

	go func(pollerer *Poller) {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case currentTime := <-tick.C:
				tags := poller.tagsForTime(currentTime)
				if len(tags) == 0 {
					continue
				}
				poller.poll(tags)
			case <-poller.done:
				return
			}

		}
	}(poller)

	return poller
}

func (poller *Poller) NewClient() *Client {
	client := &Client{
		id:     atomic.AddUint32(&poller.next, 1),
		tags:   make(map[string]uint, 0),
		poller: poller,
	}
	poller.clients[client.id] = client
	return client
}

func (poller *Poller) Close() error {
	close(poller.done)
	return nil
}

func (client *Client) Close() {
	delete(client.poller.clients, client.id)
}

func (client *Client) Poll(name string, interval uint) {
	switch {
	case interval > 0:
		client.tags[name] = interval
	default:
		delete(client.tags, name)
	}

}

func (poller *Poller) tagsForTime(currentTime time.Time) []string {
	tags := make([]string, 0)
	seconds := currentTime.Unix()
	for _, client := range poller.clients {
		for name, interval := range client.tags {
			if seconds%int64(interval) != 0 {
				continue
			}
			tags = append(tags, name)
		}

	}
	return tags
}
