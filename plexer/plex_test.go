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
package plexer

import (
	"testing"
	"time"
)

func TestPlex(test *testing.T) {
	plex := NewPlexer()
	count := 0
	plex.AddObserver("test", func(tags []string, data string) error {
		count++
		return nil
	})

	plex.Publish([]string{"what"}, "data")
	plex.Publish([]string{"what"}, "data")
	time.Sleep(time.Millisecond * 10)
	assert(test, count == 2, "publisher was called an incorrect number of times")
	plex.RemoveObserver("test")
	plex.Publish([]string{"what"}, "data")
	time.Sleep(time.Millisecond * 10)
	assert(test, count == 2, "publisher was called an incorrect number of times")
}

func assert(test *testing.T, check bool, fmt string, args ...interface{}) {
	if !check {
		test.Logf(fmt, args...)
		test.FailNow()
	}
}
