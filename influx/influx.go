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

func Insert(messageSet plexer.MessageSet) error {
	lumber.Trace("[PULSE :: INFLUX] Insert: %v...", messageSet)

	// create a set of points we will be inserting
	points := []*client.Point{}

	for _, message := range messageSet.Messages {
		// create a list of tags for each message
		tags := map[string]string{}

		// make sure to include the MessageSet's tags
		for _, tag := range append(messageSet.Tags, message.Tags...) {
			elems := strings.SplitN(tag, ":", 2)
			// only include tags with key:value format
			if len(elems) < 2 {
				continue
			}

			// insert the tag into my list of tags
			tags[elems[0]] = elems[1]
		}

		// if there
		value, err := strconv.ParseFloat(message.Data, 64)
		if err != nil {
			value = -1
		}

		// only one field per set of message tags.
		field := map[string]interface{}{message.ID: value}
		// create a point
		point, err := client.NewPoint("metrics", tags, field, time.Now())
		if err != nil {
			continue
		}
		points = append(points, point)
	}
	return writePoints("statistics", "2.days", points)
}

func writePoints(database, retain string, points []*client.Point) error {
	// Create a new point batch
	batchPoint, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        database,
		RetentionPolicy: retain,
		Precision:       "s",
	})
	for _, point := range points {
		batchPoint.AddPoint(point)
	}

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
