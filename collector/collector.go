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
		Value() int
		Flush()
		SetInterval(time.Duration)
	}

	gauge struct {
		stat     Stat
		current  int
		done     chan interface{}
		next     <-chan time.Time
		interval time.Duration
	}
)

func NewCollector(stat Stat) Collector {
	gauge := &gauge{
		stat:    stat,
		current: stat(),
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
	gauge.next = time.After(gauge.interval)
}

func (gauge *gauge) Value() int {
	return gauge.current
}

func (gauge *gauge) Flush() {
	gauge.current = 0
}

func (gauge *gauge) SetInterval(interval time.Duration) {
	gauge.interval = interval
}
