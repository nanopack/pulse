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
// Created :   9 September 2015 by Daniel Barney <daniel@nanobox.io>
//--------------------------------------------------------------------
package server

import (
	"bitbucket.org/nanobox/na-pulse/plexer"
	"github.com/influxdb/influxdb/cluster"
	"github.com/influxdb/influxdb/cmd/influxd/run"
	"github.com/influxdb/influxdb/influxql"
	"github.com/influxdb/influxdb/tsdb"
	"io"
	"strings"
	"time"
)

func (server *Server) StartInfluxd() (io.Closer, error) {
	config, err := run.NewDemoConfig()
	influx, err := run.NewServer(config, "embedded v0.9.3")
	if err != nil {
		return nil, err
	}

	err = influx.Open()
	if err != nil {
		return nil, err
	}

	server.Influx = influx

	return influx, nil
}

func (server *Server) Query(sql string) (<-chan *influxql.Result, error) {
	query, err := influxql.ParseQuery(sql)
	if err != nil {
		return nil, err
	}
	return server.Influx.QueryExecutor.ExecuteQuery(query, "testing", 1024)
}

func (server *Server) WritePoints(database, retain string, points []tsdb.Point) error {
	pointsRequest := &cluster.WritePointsRequest{
		Database:         database,
		RetentionPolicy:  retain,
		ConsistencyLevel: cluster.ConsistencyLevelAny, // TODO is this right?
		Points:           points,
	}
	return server.Influx.PointsWriter.WritePoints(pointsRequest)
}

func (server *Server) InfluxInsert(messages plexer.MessageSet) error {
	tags := make(tsdb.Tags, 0)
	for _, tag := range messages.Tags {
		elems := strings.SplitN(tag, ":", 2)
		if len(elems) < 2 {
			continue
		}

		tags[elems[0]] = elems[1]
	}

	fields := make(tsdb.Fields, len(messages.Messages))
	for _, message := range messages.Messages {
		metricId := message.Tags[0]
		fields[metricId] = message.Data
	}
	point := tsdb.NewPoint("metrics.zone", tags, fields, time.Now())
	err := server.WritePoints("statistics", "yearSingle", []tsdb.Point{point})
	return err
}
