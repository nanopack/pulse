// Package influx provides the backend for storing stats.
package influx

import (
	"fmt"
	"sort"
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
	lumber.Trace("[PULSE :: INFLUX] Querying influx: '%s'...", sql)

	c, err := influxClient()
	if err != nil {
		return nil, err
	}
	return c.Query(client.NewQuery(fmt.Sprint(sql), "statistics", "s"))
}

func Insert(messageSet plexer.MessageSet) error {
	lumber.Trace("[PULSE :: INFLUX] Insert: %+v...", messageSet)

	// create a set of points we will be inserting
	points := []*client.Point{}

	for _, message := range messageSet.Messages {
		// create a list of tags for each message
		tags := map[string]string{}

		// make sure to include the MessageSet's tags
		for _, tag := range append(messageSet.Tags, message.Tags...) {
			elems := strings.SplitN(tag, ":", 2)
			// only include tags with key:value format (all others ignored)
			if len(elems) < 2 {
				continue // we could possibly 'tag' influx entry with these single 'tags'
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
		point, err := client.NewPoint(message.ID, tags, field, time.Now())
		if err != nil {
			continue
		}
		points = append(points, point)
	}
	return writePoints("statistics", "one_day", points)
}

func writePoints(database, retain string, points []*client.Point) error {
	// Create a new point batch
	batchPoint, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        database,
		RetentionPolicy: retain,
		Precision:       "ns",
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
		Addr:    viper.GetString("influx-address"),
		Timeout: 5 * time.Second,
	})
	return clientConn, err
}

// convert map to string slice
func slicify(mappy map[string]bool) (slicey []string) {
	for k := range mappy {
		slicey = append(slicey, k)
	}
	return
}

func KeepContinuousQueriesUpToDate() error {
	lumber.Trace("[PULSE :: INFLUX] Watching continuous query...")
	c, err := influxClient()
	if err != nil {
		return err
	}
	var aggregateInterval = viper.GetInt("aggregate-interval")

	for {
		// get fields
		// todo: maybe rather than `/.*/` use `/([^a][^g][^g][^r][^e][^g][^a][^t][^e]).*/`
		// or do a `show measurements` and skip 'aggregate' or `SHOW MEASUREMENTS WITH MEASUREMENT =~ /([^a][^g][^g][^r][^e][^g][^a][^t][^e]).*/`
		cols, err := c.Query(client.NewQuery("SHOW FIELD KEYS", "statistics", "s")) // equivalent to including `FROM one_day./.*/`
		if err != nil {
			// todo: return?
			lumber.Error("Failed to show field keys from statistics - %s", err.Error())
		}

		// check tags
		groupBy, err := c.Query(client.NewQuery("SHOW TAG KEYS", "statistics", "s"))
		if err != nil {
			// todo: return?
			lumber.Error("Failed to show tag keys from statistics - %s", err.Error())
		}

		// get continuous queries
		cont, err := c.Query(client.NewQuery("SHOW CONTINUOUS QUERIES", "statistics", "s"))
		if err != nil {
			// todo: return?
			lumber.Error("Failed to show continuous queries from statistics - %s", err.Error())
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
		grp := map[string]bool{}
		for _, res := range groupBy.Results {
			for _, series := range res.Series {
				for _, val := range series.Values {
					grp[val[0].(string)] = true
				}
			}
		}
		group := slicify(grp)

		// populate current columns
		clm := map[string]bool{}
		for _, res := range cols.Results {
			for _, series := range res.Series {
				for _, val := range series.Values {
					clm[val[0].(string)] = true
				}
			}
		}
		columns := slicify(clm)

		// group columns into "mean(col) AS col"
		summary := []string{}
		for _, col := range columns {
			if col != "cpu" && col != "time" {
				summary = append(summary, fmt.Sprintf(`mean(%s) AS %s`, col, col))
			}
		}

		// sort so we don't always create new queries
		sort.Strings(summary)
		sort.Strings(group)

		// create new query string
		newQuery := `CREATE CONTINUOUS QUERY aggregate ON statistics BEGIN SELECT ` + fmt.Sprintf(strings.Join(summary, ", ")) + ` INTO statistics.one_week.aggregate FROM statistics.one_day./.*/ GROUP BY time(` + strconv.Itoa(aggregateInterval) + `m), ` + fmt.Sprintf(strings.Join(group, ", ")) + ` END`

		// if columns changed, rebuild continuous query
		if (currentQuery != newQuery) && columns != nil {
			lumber.Trace("OLD Query: %+q\n", currentQuery)
			lumber.Trace("NEW Query: %+q\n", newQuery)
			lumber.Trace("[PULSE :: INFLUX] Rebuilding continuous query...")
			r, err := c.Query(client.NewQuery(`DROP CONTINUOUS QUERY aggregate ON statistics`, "statistics", "s"))
			if err != nil {
				lumber.Error("Failed to drop continuous queries - %+v - %+v", r, err)
			}
			lumber.Trace("New Query: %+s", newQuery)
			r, err = c.Query(client.NewQuery(newQuery, "statistics", "s"))
			if err != nil {
				lumber.Error("Failed to create continuous query - %+v - %+v", r, err)
			}
		}

		<-time.After(time.Duration(aggregateInterval) * time.Minute)
	}
}
