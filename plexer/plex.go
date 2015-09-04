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
	"bitbucket.org/nanobox/na-pulse/server"
	"errors"
)

var (
	MissingPublisher = errors.New("A publisher is needed")
)

type (
	Plexer struct {
		observers map[string]server.Publisher
	}
)

func NewPlexer() *Plexer {
	plex := &Plexer{
		observers: make(map[string]server.Publisher, 0),
	}

	return plex
}

func (plex *Plexer) AddObserver(name string, observer server.Publisher) {
	plex.observers[name] = observer
}

func (plex *Plexer) RemoveObserver(name string) {
	delete(plex.observers, name)
}

func (plex *Plexer) Publish(tags []string, data string) error {
	for _, observer := range plex.observers {
		go observer(tags, data)
	}
	return nil
}
