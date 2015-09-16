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
package collector

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestGauge(test *testing.T) {
	count := 0
	random := func() float64 {
		count++
		return float64(rand.Int() % 100)
	}
	collect := NewPointCollector(random)

	collect.Start()
	collect.SetInterval(time.Millisecond)
	time.Sleep(time.Millisecond * 10)
	collect.Stop()
	assert(test, count > 5, "collector was not called enough times %v", count)

	collect = NewPointCollector(RunningAverage(random, 100))
	collect.SetInterval(time.Millisecond)
	collect.Start()
	time.Sleep(time.Millisecond * 100)
	collect.Stop()
	assert(test, math.Abs(collect.Values()[""]-50) < 10, "not a very good random number generator %v", collect.Values())
}

func assert(test *testing.T, check bool, fmt string, args ...interface{}) {
	if !check {
		test.Logf(fmt, args...)
		test.FailNow()
	}
}
