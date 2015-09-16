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
// Created :   8 September 2015 by Daniel Barney <daniel@nanobox.io>
//--------------------------------------------------------------------
package collector

type (
	DataSet func() map[string]float64
	set     struct {
		Collect

		set     DataSet
		current map[string]float64
	}
)

func NewSetCollector(stats DataSet) Collector {
	set := &set{
		set: stats,
	}
	set.collectValue()
	set.CollectFun = set.collectValue

	return set
}

func (set *set) Values() map[string]float64 {
	return set.current
}

func (set *set) Flush() {
	set.current = make(map[string]float64, 0)
}

func (set *set) collectValue() {
	set.current = set.set()
}
