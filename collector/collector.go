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
// Created :   31 August 2015 by Daniel Barney <daniel@nanobox.io>
//--------------------------------------------------------------------
package collector

import (
	"time"
)

type (
	Stat      func() int
	Collector interface {
		Stop()
		Start()
		Values() map[string]int
		Flush()
		SetInterval(time.Duration)
		OverrideInterval(time.Duration, time.Duration)
	}

	collector struct {
		done     chan interface{}
		next     <-chan time.Time
		revert   chan bool
		interval time.Duration
		override time.Duration
		collect  func()
	}
)

func (collector *collector) SetInterval(interval time.Duration) {
	collector.interval = interval
	collector.reset()
}

func (collector *collector) OverrideInterval(newInterval time.Duration, howLong time.Duration) {
	if collector.override != 0 {
		close(collector.revert)
	}
	collector.override = newInterval
	collector.revert = make(chan bool)
	collector.reset()
	go func() {
		select {
		case <-time.After(howLong):
			collector.override = 0
			collector.reset()
		case <-collector.revert:
			return
		}
	}()
}

func (collector *collector) reset() {
	switch {
	case collector.override != 0:
		collector.next = time.After(collector.override)
	default:
		collector.next = time.After(collector.interval)
	}

}

func (collector *collector) Stop() {
	if collector.done != nil {
		close(collector.done)
		collector.done = nil
	}
}

func (collector *collector) Start() {
	if collector.done == nil {
		collector.reset()
		collector.done = make(chan interface{})
		go func() {
			for {
				select {
				case <-collector.done:
					return
				case <-collector.next:
					collector.reset()
					collector.collect()
				}
			}
		}()
	}
}
