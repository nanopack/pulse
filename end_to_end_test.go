// -*- mode: go; tab-width: 2; indent-tabs-mode: 1; st-rulers: [70] -*-
// vim: ts=4 sw=4 ft=lua noet
//--------------------------------------------------------------------
// @author Daniel Barney <daniel@nanobox.io>
// Copyright (C) Pagoda Box, Inc - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly
// prohibited. Proprietary and confidential
//
// @doc
//
// @end
// Created :   4 September 2015 by Daniel Barney <daniel@nanobox.io>
//--------------------------------------------------------------------
package main

import (
	"bitbucket.org/nanobox/na-pulse/collector"
	"bitbucket.org/nanobox/na-pulse/plexer"
	"bitbucket.org/nanobox/na-pulse/relay"
	"bitbucket.org/nanobox/na-pulse/server"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var address = "127.0.0.1:1234"

func TestEndToEnd(test *testing.T) {
	wait := sync.WaitGroup{}
	server, err := server.Listen(address, func(messages plexer.MessageSet) error {
		wait.Add(-len(messages.Messages))
		return nil
	})

	assert(test, err == nil, "unable to listen %v", err)
	defer server.Close()

	relay, err := relay.NewRelay(address, "relay.station.1")
	assert(test, err == nil, "unable to connect to server %v", err)
	defer relay.Close()

	cpuCollector := randCollector()
	relay.AddCollector("cpu", cpuCollector)

	ramCollector := randCollector()
	relay.AddCollector("ram", ramCollector)

	diskCollector := randCollector()
	relay.AddCollector("disk", diskCollector)
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
	collect := collector.NewGauge(rand.Int)
	collect.SetInterval(time.Millisecond * 10)
	return collect
}

func assert(test *testing.T, check bool, fmt string, args ...interface{}) {
	if !check {
		test.Logf(fmt, args...)
		test.FailNow()
	}
}
