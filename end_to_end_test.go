package main

import (
	"github.com/nanopack/pulse/collector"
	"github.com/nanopack/pulse/plexer"
	"github.com/nanopack/pulse/relay"
	"github.com/nanopack/pulse/server"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var address = "127.0.0.1:1234"

func TestEndToEnd(test *testing.T) {
	wait := sync.WaitGroup{}
	err := server.Listen(address, func(messages plexer.MessageSet) error {
		wait.Add(-len(messages.Messages))
		return nil
	})

	if err != nil {
		test.Errorf("unable to listen %v", err)
		return
	}

	relay, err := relay.NewRelay(address, "relay.station.1")
	if err != nil {
		test.Errorf("unable to connect to server %v", err)
		return
	}

	defer relay.Close()

	cpuCollector := randCollector()
	relay.AddCollector("cpu", []string{"hi", "how", "are:you"}, cpuCollector)

	ramCollector := randCollector()
	relay.AddCollector("ram", nil, ramCollector)

	diskCollector := randCollector()
	relay.AddCollector("disk", nil, diskCollector)
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

}

func randCollector() collector.Collector {
	collect := collector.NewPointCollector(rand.Float64)
	collect.SetInterval(time.Millisecond * 10)
	return collect
}

func assert(test *testing.T, check bool, fmt string, args ...interface{}) {
	if !check {
		test.Logf(fmt, args...)
		test.FailNow()
	}
}
