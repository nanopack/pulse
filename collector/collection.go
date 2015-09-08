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
	Collection func() map[string]int
	collection struct {
		collector

		collection Collection
		current    map[string]int
	}
)

func NewCollection(stats Collection) Collector {
	collection := &collection{
		collection: stats,
	}
	collection.collectValue()
	collection.collect = collection.collectValue

	return collection
}

func (collection *collection) Values() map[string]int {
	return collection.current
}

func (collection *collection) Flush() {
	collection.current = make(map[string]int, 0)
}

func (collection *collection) collectValue() {
	collection.current = collection.collection()
}
