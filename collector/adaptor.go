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
	"math"
)

func MutedAverage(stat Stat, amount int) Stat {
	current := 0
	return func() int {
		current += stat() / amount
		return current
	}
}

func Average(stat Stat) Stat {
	count := 0
	current := 0
	return func() int {
		// has to be a better way to do this
		current = (current*count + stat()) / (count + 1)
		count++
		return current
	}
}

func RunningAverage(stat Stat, length int) Stat {
	values := make([]int, length)
	idx := 0
	return func() int {
		values[idx%length] = stat()
		idx++
		count := 0
		value := 0
		for _, val := range values {
			value += val
			count++
		}
		count = int(math.Min(float64(count), float64(idx)))
		if count == 0 {
			return 0
		}
		return value / count
	}
}
