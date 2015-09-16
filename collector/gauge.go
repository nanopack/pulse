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

type (
	DataPoint func() float64
	gauge     struct {
		Collect

		stat    DataPoint
		current float64
	}
)

func NewPointCollector(stat DataPoint) Collector {
	gauge := &gauge{
		stat: stat,
	}
	gauge.collectValue()
	gauge.CollectFun = gauge.collectValue

	return gauge
}

func (gauge *gauge) Values() map[string]float64 {
	return map[string]float64{"": gauge.current}
}

func (gauge *gauge) Flush() {
	gauge.current = 0
}

func (gauge *gauge) collectValue() {
	gauge.current = gauge.stat()
}
