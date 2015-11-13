package server

import (
	"github.com/influxdb/influxdb/cluster"
	"github.com/influxdb/influxdb/cmd/influxd/run"
	"github.com/influxdb/influxdb/influxql"
	"github.com/influxdb/influxdb/models"
	"github.com/nanopack/pulse/plexer"
	"io"
	"strconv"
	"strings"
	"time"
)

func (server *Server) StartInfluxd() (io.Closer, error) {
	config, err := run.NewDemoConfig()
	type BuildInfo struct {
		Version string
		Commit  string
		Branch  string
		Time    string
	}
	build := run.BuildInfo{
		Version: "v0.9.3",
		Commit:  "sha hash",
		Branch:  "master",
		Time:    "now",
	}
	influx, err := run.NewServer(config, &build)
	if err != nil {
		return nil, err
	}

	err = influx.Open()
	if err != nil {
		return nil, err
	}

	server.Influx = influx
	// wait for it to set it self up.
	time.Sleep(time.Second * 2)
	return influx, nil
}

func (server *Server) Query(sql string) (<-chan *influxql.Result, error) {
	query, err := influxql.ParseQuery(sql)
	if err != nil {
		return nil, err
	}
	return server.Influx.QueryExecutor.ExecuteQuery(query, "statistics", 1024)
}

func (server *Server) WritePoints(database, retain string, points []models.Point) error {
	pointsRequest := &cluster.WritePointsRequest{
		Database:         database,
		RetentionPolicy:  retain,
		ConsistencyLevel: cluster.ConsistencyLevelAny, // TODO is this right?
		Points:           points,
	}
	return server.Influx.PointsWriter.WritePoints(pointsRequest)
}

func (server *Server) InfluxInsert(messages plexer.MessageSet) error {
	tags := make(models.Tags, 0)
	for _, tag := range messages.Tags {
		elems := strings.SplitN(tag, ":", 2)
		if len(elems) < 2 {
			continue
		}

		tags[elems[0]] = elems[1]
	}

	fields := make(models.Fields, len(messages.Messages))
	for _, message := range messages.Messages {
		metricId := message.Tags[0]
		value, err := strconv.ParseFloat(message.Data, 64)
		if err != nil {
			value = -1
		}
		fields[metricId] = value
	}
	point, err := models.NewPoint("metrics", tags, fields, time.Now())
	if err != nil {
		return err
	}
	return server.WritePoints("statistics", "2.days", []models.Point{point})
}
