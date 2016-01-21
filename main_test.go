package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nanopack/pulse/plexer"
	"github.com/nanopack/pulse/relay"
	"github.com/nanopack/pulse/server"
)

var address = "127.0.0.1:1234"
var wait = sync.WaitGroup{}

var messages = []plexer.MessageSet{}

func TestMain(m *testing.M) {
	err := server.Listen(address, func(msgSet plexer.MessageSet) error {
		messages = append(messages, msgSet)
		wait.Add(-len(msgSet.Messages))
		return nil
	})

	if err != nil {
		panic(fmt.Sprintf("unable to listen %v", err))
		return
	}
	rtn := m.Run()
	os.Exit(rtn)
}

func TestEndToEnd(test *testing.T) {
	relay, err := relay.NewRelay(address, "relay.station.1")
	if err != nil {
		test.Errorf("unable to connect to server %v", err)
		return
	}
	defer relay.Close()

	relay.AddCollector("cpu", []string{"hi", "how", "are:you"}, randCollector())

	relay.AddCollector("ram", nil, randCollector())

	relay.AddCollector("disk", nil, randCollector())
	time.Sleep(time.Millisecond * 100)
	wait.Add(1)
	server.Poll([]string{"disk"})
	wait.Wait()

	wait.Add(2)
	server.Poll([]string{"ram", "cpu"})
	wait.Wait()

	wait.Add(3)
	server.Poll([]string{"ram", "cpu", "disk"})
	wait.Wait()

	if len(messages) != 3 {
		test.Errorf("Expected to recieve 3 messages but instead got %d", len(messages))
	}
	messages = []plexer.MessageSet{}
}

func randCollector() relay.Collector {
	return relay.NewPointCollector(rand.Float64)
}
