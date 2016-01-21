package influx

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/jcelliott/lumber"
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
	lumber.Trace("[PULSE :: INFLUX] Insert: %v...", messages)
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
		Database:        database,
		RetentionPolicy: retain,
		Precision:       "s",
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

func KeepContinuousQueriesUpToDate() error {
	lumber.Trace("[PULSE :: INFLUX] Watching continuous query...")
	c, err := influxClient()
	if err != nil {
		return err
	}
	var aggregate_interval = viper.GetInt("aggregate_interval")

	for {
		// get fields
		cols, err := c.Query(client.NewQuery("SHOW FIELD KEYS FROM \"2.days\".\"metrics\"", "statistics", "s"))
		if err != nil {
			panic(err)
		}

		// check tags
		groupBy, err := c.Query(client.NewQuery("SHOW TAG KEYS FROM \"2.days\".\"metrics\"", "statistics", "s"))
		if err != nil {
			panic(err)
		}

		// get continuous queries
		cont, err := c.Query(client.NewQuery("SHOW CONTINUOUS QUERIES", "statistics", "s"))
		if err != nil {
			panic(err)
		}

		// get current query
		var currentQuery string
		for _, res := range cont.Results {
			for _, series := range res.Series {
				if series.Name == "statistics" {
					for _, val := range series.Values {
						if val[0].(string) == "aggregate" {
							currentQuery = val[1].(string)
						}
					}
				}
			}
		}

		// populate current tags
		group := []string{}
		for _, res := range groupBy.Results {
			for _, series := range res.Series {
				if series.Name == "metrics" {
					for _, val := range series.Values {
						group = append(group, val[0].(string))
					}
				}
			}
		}

		// populate current columns
		columns := []string{}
		for _, res := range cols.Results {
			for _, series := range res.Series {
				if series.Name == "metrics" {
					for _, val := range series.Values {
						columns = append(columns, val[0].(string))
					}
				}
			}
		}

		// group columns into "mean(col) AS col"
		summary := []string{}
		for _, col := range columns {
			if col != "cpu" && col != "time" {
				summary = append(summary, fmt.Sprintf(`mean(%s) AS "%s"`, col, col))
			}
		}

		// create new query string
		newQuery := `CREATE CONTINUOUS QUERY aggregate ON statistics BEGIN SELECT ` + fmt.Sprintf(strings.Join(summary, ", ")) + ` INTO statistics."1.week".metrics FROM statistics."2.days".metrics GROUP BY time(` + strconv.Itoa(aggregate_interval) + `m), ` + fmt.Sprintf(strings.Join(group, ", ")) + ` END`

		// if columns changed, rebuild continuous query
		if (currentQuery != newQuery) && columns != nil {
			lumber.Trace("[PULSE :: INFLUX] Rebuilding continuous query...")
			r, err := c.Query(client.NewQuery(`DROP CONTINUOUS QUERY aggregate ON statistics`, "statistics", "s"))
			if err != nil {
				fmt.Printf("ERROR: %+v, %+v\n", r, err)
			}

			r, err = c.Query(client.NewQuery(newQuery, "statistics", "s"))
			if err != nil {
				fmt.Printf("ERROR: %+v, %+v\n", r, err)
			}
		}

		<-time.After(time.Duration(aggregate_interval) * time.Minute)
	}
}
