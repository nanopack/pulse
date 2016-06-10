package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/nanopack/pulse/influx"
)

type (
	point struct {
		Time  int64   `json:"time"`
		Value float64 `json:"value"`
	}
)

// return a list of fields
func keysRequest(res http.ResponseWriter, req *http.Request) {
	cols, err := influx.Query("SHOW FIELD KEYS FROM one_day./.*/") // or show measurements? - aggregate
	if err != nil {
		panic(err)
	}

	columns := make(map[string]struct{})
	for _, result := range cols.Results {
		for _, series := range result.Series {
			for _, val := range series.Values {
				columns[val[0].(string)] = struct{}{}
			}
		}
	}

	var fields []string
	for k, _ := range columns {
		fields = append(fields, k)
	}

	writeBody(fields, res, http.StatusOK, req)
}

// return a list of tags
func tagsRequest(res http.ResponseWriter, req *http.Request) {
	groupBy, err := influx.Query("SHOW TAG KEYS FROM one_day./.*/")
	if err != nil {
		panic(err)
	}

	columns := make(map[string]struct{})
	for _, result := range groupBy.Results {
		for _, series := range result.Series {
			for _, val := range series.Values {
				columns[val[0].(string)] = struct{}{}
			}
		}
	}
	var tags []string
	for k, _ := range columns {
		tags = append(tags, k)
	}

	writeBody(tags, res, http.StatusOK, req)
}

// fetches the latest stat for either a single filter (eg. host) or the average of multiple
func latestStat(res http.ResponseWriter, req *http.Request) {
	// todo: prevent sql(like)-injection (start secure, its their own private stat db otherwise)

	// verb is what to get to the stat (mean, min, max)
	verb := req.URL.Query().Get("verb")
	if verb == "" {
		verb = "mean" // default to average (avg of [x] is x)
	}

	// limit is how many to limit, and will be derived from the length of specified
	// array: (`host=a&host=b&limit=host` limit would be 2)
	var limit int
	filter := req.URL.Query().Get("limit")

	if filter == "" || len(req.URL.Query()[filter]) == 0 {
		limit = 1 // default to 1
	} else {
		limit = len(req.URL.Query()[filter])
	}

	// stat is the stat we are selecting
	stat := req.URL.Query().Get(":stat")
	query := fmt.Sprintf(`SELECT %v("%v") FROM "%v"`, verb, stat, stat)

	filters := []string{}
	for key, val := range req.URL.Query() {
		if key == ":stat" || key == "limit" || key == "verb" {
			continue
		}
		// if there are multiple values, "OR" them, and save as one filter element
		// allows `host=a&host=b` in query
		if len(val) > 0 {
			var tfil []string
			for i := range val {
				tfil = append(tfil, fmt.Sprintf("\"%s\" = '%s'", key, val[i]))
			}
			filters = append(filters, fmt.Sprintf("(%s)", strings.Join(tfil, " OR ")))
			continue
		}
		filters = append(filters, fmt.Sprintf("\"%s\" = ''", key))
	}

	// filter by filters
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}

	// grab the latest
	query = fmt.Sprintf("%s ORDER BY time DESC LIMIT %d", query, limit)

	records, err := influx.Query(query)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusInternalServerError, req)
		return
	}

	if len(records.Results[0].Series) != 1 {
		writeBody(apiError{ErrorString: "Not Found"}, res, http.StatusNotFound, req)
		return
	}

	result := point{}
	val := records.Results[0].Series[0].Values[0]
	result.Time, _ = val[0].(json.Number).Int64()
	result.Value, _ = val[1].(json.Number).Float64()

	writeBody(result, res, http.StatusOK, req)
}

// fetches the historic 1h averages for a single stat
func hourlyStat(res http.ResponseWriter, req *http.Request) {
	// verb is what to get to the stat (mean, min, max)
	verb := req.URL.Query().Get("verb")
	if verb == "" {
		verb = "mean" // default to average (avg of [x] is x)
	}

	// start and stop should be an int and one of u,ms,s,m,h,d,w
	// start is how far back in time to start (will be subtracted from now())
	start := req.URL.Query().Get("start")
	if start == "" {
		start = "1d"
	}

	// stop is how far back in time to stop (will be subtracted from now())
	stop := req.URL.Query().Get("stop")
	if stop == "" {
		stop = "0s"
	}

	// stat is the stat we are selecting
	stat := req.URL.Query().Get(":stat")
	query := fmt.Sprintf(`SELECT %s("%v") FROM one_week.aggregate`, verb, stat)

	filters := []string{}
	for key, val := range req.URL.Query() {
		if key == ":stat" || key == "verb" || key == "start" || key == "stop" {
			continue
		}
		// if there are multiple values, "OR" them, and save as one filter element
		// allows `host=a&host=b` in query
		if len(val) > 0 {
			var tfil []string
			for i := range val {
				tfil = append(tfil, fmt.Sprintf("\"%s\" = '%s'", key, val[i]))
			}
			filters = append(filters, fmt.Sprintf("(%s)", strings.Join(tfil, " OR ")))
			continue
		}
		filters = append(filters, fmt.Sprintf("\"%s\" = ''", key))
	}

	// set time filter
	filters = append(filters, fmt.Sprintf("time > now() - %v", start))
	filters = append(filters, fmt.Sprintf("time < now() - %v", stop))

	// filter by filters
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}

	// grab the hourly averages ('group by' should aggregate multiple filters)
	query = fmt.Sprintf("%s GROUP BY time(1h) fill(none)", query) //  OR fill(0) to not return 0 for empty values

	records, err := influx.Query(query)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusInternalServerError, req)
		return
	}

	if len(records.Results[0].Series) != 1 {
		writeBody(apiError{ErrorString: "Not Found"}, res, http.StatusNotFound, req)
		return
	}

	result := make([]point, len(records.Results[0].Series[0].Values))
	for i, values := range records.Results[0].Series[0].Values {
		// todo: do i need to multiply time by 1000 (to return nanoseconds?)?
		result[i].Time, _ = values[0].(json.Number).Int64()
		result[i].Time = result[i].Time * 1000
		result[i].Value, _ = values[1].(json.Number).Float64()
	}

	writeBody(result, res, http.StatusOK, req)
}

// fetches the historic values for a single stat and averages each hour (useful for spotting trends - radial)
func dailyStat(res http.ResponseWriter, req *http.Request) {
	// verb is what to get to the stat (mean, min, max)
	verb := req.URL.Query().Get("verb")
	if verb == "" {
		verb = "mean" // default to average (avg of [x] is x)
	}

	// start and stop should be an int and one of u,ms,s,m,h,d,w
	// start is how far back in time to start (will be subtracted from now())
	start := req.URL.Query().Get("start")
	if start == "" {
		start = "7d"
	}

	// stop is how far back in time to stop (will be subtracted from now())
	stop := req.URL.Query().Get("stop")
	if stop == "" {
		stop = "0s"
	}

	// stat is the stat we are selecting
	stat := req.URL.Query().Get(":stat")
	query := fmt.Sprintf(`SELECT %s("%v") FROM one_week.aggregate`, verb, stat)

	filters := []string{}
	for key, val := range req.URL.Query() {
		if key == ":stat" || key == "verb" || key == "start" || key == "stop" {
			continue
		}
		// if there are multiple values, "OR" them, and save as one filter element
		// allows `host=a&host=b` in query
		if len(val) > 0 {
			var tfil []string
			for i := range val {
				tfil = append(tfil, fmt.Sprintf("\"%s\" = '%s'", key, val[i]))
			}
			filters = append(filters, fmt.Sprintf("(%s)", strings.Join(tfil, " OR ")))
			continue
		}
		filters = append(filters, fmt.Sprintf("\"%s\" = ''", key))
	}

	// set time filter
	filters = append(filters, fmt.Sprintf("time > now() - %v", start))
	filters = append(filters, fmt.Sprintf("time < now() - %v", stop))

	// filter by filters
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}

	// grab the hourly averages
	query = fmt.Sprintf("%s GROUP BY time(15m) fill(none)", query)

	records, err := influx.Query(query)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusInternalServerError, req)
		return
	}
	if len(records.Results[0].Series) != 1 {
		writeBody(apiError{ErrorString: "Not Found"}, res, http.StatusNotFound, req)
		return
	}

	result := make(map[string]float64, 24*4)
	counter := make(map[string]int, 24*4)

	for _, values := range records.Results[0].Series[0].Values {
		valTime, _ := values[0].(json.Number).Int64()
		valUnix := time.Unix(valTime, 0)
		hour, minute, _ := valUnix.Clock()
		minute = int(math.Floor(float64(minute/15.0)) * 15)
		id := fmt.Sprintf("%v:%v", hour, minute)
		valData, _ := values[1].(json.Number).Float64()

		// keep a running average instead of a total
		result[id] = ((result[id] * float64(counter[id])) + valData) / float64(counter[id]+1)
		counter[id] += 1
	}

	writeBody(result, res, http.StatusOK, req)
}
