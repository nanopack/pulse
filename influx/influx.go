package influx

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/spf13/viper"

	"github.com/nanopack/pulse/plexer"
)

var clientConn client.Client

func Query(sql string) (*client.Response, error) {
	c, err := influxClient()
	if err != nil {
		return nil, err
	}
	return c.Query(client.NewQuery(fmt.Sprint(sql), "statistics", "s"))
}

func Insert(messages plexer.MessageSet) error {
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
	point, err := client.NewPoint("metrics", tags, fields, time.Now())
	if err != nil {
		return err
	}
	return writePoints("statistics", "2.days", *point)
}

func writePoints(database, retain string, point client.Point) error {
	// Create a new point batch
	batchPoint, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		RetentionPolicy: retain,
		Precision: "s",
	})

	batchPoint.AddPoint(&point)
	c, err := influxClient()
	if err != nil {
		return err
	}
	return c.Write(batchPoint)
}

func influxClient() (client.Client, error) {
	var err error

	if clientConn != nil {
		return clientConn, nil
	}
	clientConn, err = client.NewHTTPClient(client.HTTPConfig{
		Addr: viper.GetString("influx_address"),
	})
	return clientConn, err
}