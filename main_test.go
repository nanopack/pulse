package main_test

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

var address = "127.0.0.1:8080"
var wait = sync.WaitGroup{}

var messages = []plexer.MessageSet{}

func TestMain(m *testing.M) {
	fmt.Println("Starting to listen...")
	err := server.Listen(address, func(msgSet plexer.MessageSet) error {
		messages = append(messages, msgSet)
		wait.Add(-len(msgSet.Messages))
		return nil
	})
	fmt.Println("Server listening...")

	if err != nil {
		panic(fmt.Sprintf("unable to listen %v", err))
	}
	rtn := m.Run()
	os.Exit(rtn)
}

func TestEndToEnd(test *testing.T) {
	fmt.Println("Testing end to end...")
	relay, err := relay.NewRelay(address, "relay.station.1")
	if err != nil {
		test.Errorf("unable to connect to server %v", err)
		return
	}
	defer relay.Close()
	fmt.Println("New relay created...")

	relay.AddCollector("cpu", []string{"hi", "how", "are:you"}, randCollector())
	fmt.Println("CPU relay added...")

	relay.AddCollector("ram", nil, randCollector())
	fmt.Println("RAM relay added...")

	relay.AddCollector("disk", nil, randCollector())
	fmt.Println("DISK relay added...")

	time.Sleep(time.Millisecond * 100)

	fmt.Println("DISK polling...")
	wait.Add(1)
	server.Poll([]string{"disk"})
	fmt.Println("DISK polled")
	wait.Wait()

	fmt.Println("RAM, CPU polling...")
	wait.Add(2)
	server.Poll([]string{"ram", "cpu"})
	fmt.Println("RAM, CPU polled")
	wait.Wait()

	fmt.Println("ALL polling...")
	wait.Add(3)
	server.Poll([]string{"ram", "cpu", "disk"})
	fmt.Println("ALL polled")
	wait.Wait()

	if len(messages) != 3 {
		test.Errorf("Expected to recieve 3 messages but instead got %d", len(messages))
	}
	messages = []plexer.MessageSet{}
}

func randCollector() relay.Collector {
	return relay.NewPointCollector(rand.Float64)
}
