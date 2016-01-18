package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"

	"github.com/nanopack/pulse/plexer"
)

func (server *Server) Query(sql string) (*influx.Response, error) {
	return server.influxClient.Query(influx.NewQuery(fmt.Sprint(sql), "statistics", "s"))
}

func (server *Server) writePoints(database, retain string, point influx.Point) error {
	// Create a new point batch
	batchPoint, _ := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  database,
		RetentionPolicy: retain,
		Precision: "s",
	})

	batchPoint.AddPoint(&point)
	return server.influxClient.Write(batchPoint)
}

func (server *Server) InfluxInsert(messages plexer.MessageSet) error {
	tags := make(map[string]string, 0)
	for _, tag := range messages.Tags {
		elems := strings.SplitN(tag, ":", 2)
		if len(elems) < 2 {
			continue
		}

		tags[elems[0]] = elems[1]
	}

	fields := make(map[string]interface{}, len(messages.Messages))
	for _, message := range messages.Messages {
		metricId := message.Tags[0]
		value, err := strconv.ParseFloat(message.Data, 64)
		if err != nil {
			value = -1
		}
		fields[metricId] = value
	}

	// create a point
	point, err := influx.NewPoint("metrics", tags, fields, time.Now())
	if err != nil {
		return err
	}
	return server.writePoints("statistics", "2.days", *point)
}
