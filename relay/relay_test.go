package relay_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nanopack/pulse/plexer"
	"github.com/nanopack/pulse/relay"
	"github.com/nanopack/pulse/server"
)

var serverAddr = "127.0.0.1:9899"
var testRelay *relay.Relay

func stdoutPublisher(messages plexer.MessageSet) error {
	// immitation batch
	for _, message := range messages.Messages {
		tags := map[string]string{}
		for _, tag := range append(messages.Tags, message.Tags...) {
			elems := strings.SplitN(tag, ":", 2)
			// only keep key:value tags
			if len(elems) < 2 {
				continue
			}

			// save as tags[key] = value
			tags[elems[0]] = elems[1]
		}

		fmt.Printf("BATCH : %s, %s, %s\n", message.ID, tags, message.Data)
	}

	// immitation single
	for _, message := range messages.Messages {
		message.Tags = append(message.Tags, messages.Tags...)
		fmt.Printf("SINGLE: %+q, %s\n", append(message.Tags, message.ID), message.Data)
	}

	return nil
}

func TestMain(m *testing.M) {
	go server.StartPolling(nil, nil, 1*time.Second, nil)

	err := server.Listen(serverAddr, stdoutPublisher)
	if err != nil {
		fmt.Printf("Failed to start server - %s\n", err)
		return
	}

	time.Sleep(time.Second)

	os.Exit(m.Run())
}

func TestCollectors(t *testing.T) {
	testRelay, err := relay.NewRelay(serverAddr, "test_client")
	if err != nil {
		fmt.Printf("Failed to create relay - %s\n", err)
		return
	}

	ramCollector := relay.NewPointCollector(func() float64 {
		return 25.0
	})

	fmt.Println("Adding ram collector")
	if err := testRelay.AddCollector("ram", []string{"guy"}, ramCollector); err != nil {
		t.Errorf("Failed to add ram collector - %s\n", err)
		t.FailNow()
	}

	diskCollector := relay.NewSetCollector(func() map[string]float64 {
		return map[string]float64{"disk": 50.0}
	})

	fmt.Println("Adding disk collector")
	if err := testRelay.AddCollector("disk", []string{"guy"}, diskCollector); err != nil {
		t.Errorf("Failed to add disk collector - %s\n", err)
		t.FailNow()
	}

	if err := testRelay.AddCollector("disk", []string{"guy"}, diskCollector); err == nil {
		t.Errorf("Failed to fail addding disk collector - %s\n", err)
		t.FailNow()
	}

	if err := testRelay.AddCollector("_connected", []string{"guy"}, diskCollector); err == nil {
		t.Errorf("Failed to fail addding reserved collector - %s\n", err)
		t.FailNow()
	}

	fmt.Println("INFO - ", testRelay.Info())
	time.Sleep(3 * time.Second)

	fmt.Println("Removing collector")
	testRelay.RemoveCollector("ram-used")
	testRelay.RemoveCollector("ram")

	time.Sleep(1 * time.Second)

	testRelay.Close()
}
