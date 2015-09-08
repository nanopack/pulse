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

	gauge struct {
		stat     Stat
		current  int
		done     chan interface{}
		next     <-chan time.Time
		revert   chan bool
		interval time.Duration
		override time.Duration
	}
)

func NewCollector(stat Stat) Collector {
	gauge := &gauge{
		stat:     stat,
		current:  stat(),
		override: 0,
	}

	return gauge
}

func (gauge *gauge) Stop() {
	if gauge.done != nil {
		close(gauge.done)
		gauge.done = nil
	}
}

func (gauge *gauge) Start() {
	if gauge.done == nil {
		gauge.reset()
		gauge.done = make(chan interface{})
		go func() {
			for {
				select {
				case <-gauge.done:
					return
				case <-gauge.next:
					gauge.reset()
					gauge.current = gauge.stat()
				}
			}
		}()
	}
}

func (gauge *gauge) reset() {
	switch {
	case gauge.override != 0:
		gauge.next = time.After(gauge.override)
	default:
		gauge.next = time.After(gauge.interval)
	}

}

func (gauge *gauge) Values() map[string]int {
	return map[string]int{"": gauge.current}
}

func (gauge *gauge) Flush() {
	gauge.current = 0
}

func (gauge *gauge) SetInterval(interval time.Duration) {
	gauge.interval = interval
}

func (gauge *gauge) OverrideInterval(newInterval time.Duration, howLong time.Duration) {
	if gauge.override != 0 {
		close(gauge.revert)
	}
	gauge.override = newInterval
	gauge.revert = make(chan bool)
	go func() {
		select {
		case <-time.After(howLong):
			gauge.override = 0
		case <-gauge.revert:
			return
		}
	}()
}
