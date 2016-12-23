package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/nanopack/pulse/influx"
)

type (
	point struct {
		Time  int64   `json:"time"`
		Value float64 `json:"value"`
	}

	dayPoint struct {
		Time  string  `json:"time"`
		Value float64 `json:"value"`
	}
)

// return a list of fields
func keysRequest(res http.ResponseWriter, req *http.Request) {
	cols, err := influx.Query("SHOW FIELD KEYS FROM one_day./.*/") // or show measurements? - aggregate
	if err != nil {
		// todo: don't panic, we can handle this
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

	// multiFilter allows for information to be given when multiple same-key filters are applied
	// on a query where verb is 'none'. eg. `host=a&host=b&verb=none` would return 2 results and
	// that include the host info.
	multiFilter := []string{}

	// default how many responses to limit
	limit := 1

	// filters holds the influxDB language for the WHERE clause
	filters := []string{}
	for key, val := range req.URL.Query() {
		if key == ":stat" || key == "verb" || key == "start" || key == "stop" || key == "backfill" || key == "x-auth-token" || key == "X-AUTH-TOKEN" {
			continue
		}
		if len(val) > 1 {
			multiFilter = append(multiFilter, key)
			// if filter applied number is greater than the limit, set limit to that
			if len(val) > limit {
				limit = len(val)
			}
		}
		// if there are multiple values, "OR" them, and save as one filter element
		// allows `host=a&host=b` in query
		if len(val) > 0 {
			var tfil []string
			for i := range val {
				// todo: does req.Url.Query()[intkey] return an int or is it always a string?
				tfil = append(tfil, fmt.Sprintf("\"%s\" = '%s'", key, val[i]))
			}
			filters = append(filters, fmt.Sprintf("(%s)", strings.Join(tfil, " OR ")))
			continue
		}
		filters = append(filters, fmt.Sprintf("\"%s\" = ''", key))
	}

	// stat is the stat we are selecting
	stat := req.URL.Query().Get(":stat")
	var query string
	if verb == "none" {
		query = fmt.Sprintf(`SELECT "%s"`, stat)
		// return multiFilter info
		if len(multiFilter) > 0 {
			query = fmt.Sprintf(`%s, "%s"`, query, strings.Join(multiFilter, "\", \""))
		}
	} else {
		query = fmt.Sprintf(`SELECT %s("%s")`, verb, stat)
	}

	// add the FROM to the query
	query = fmt.Sprintf(`%s FROM "%s"`, query, stat)

	// grab the latest chunk (poll interval used so we don't have too many results)
	// allow missing 1 stat update (we have to sacrifice accuracy because influx
	// acts eventually consistent and doesn't return the data in a predictable,
	// timely manner)
	filters = append(filters, fmt.Sprintf("time > now() - %ds", viper.GetInt("poll-interval")*2))

	// filter by filters
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}

	// if aggregate function used, group by time and limit to 1
	if verb != "none" {
		limit = 1
		query = fmt.Sprintf("%s GROUP BY time(%ds) FILL(none)", query, viper.GetInt("poll-interval"))
	}

	// grab the latest available
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

	// if there are multiple values (if a filter was applies more than 1x)
	if len(records.Results[0].Series[0].Values) > 1 {
		// create the master result map (will have key, val)
		resulto := make([]map[string]interface{}, 0)

		// loop through the values ([][]interface{})
		for _, val := range records.Results[0].Series[0].Values {
			// create a temporary result for holding values by column name
			tRes := make(map[string]interface{})

			// loop through nested slice
			for i := range val {
				// time will be at index 0 unless influx changes things
				// eg. (select "value"..; returns "time, value")
				if i == 0 {
					// make time uniform
					timestamp, _ := val[i].(json.Number).Int64()
					tRes["time"] = timestamp * 1000
					continue
				}
				// our value will be at index 1
				if i == 1 {
					// make value generic
					tRes["value"] = val[i]
					continue
				}
				// assign value to column name
				tRes[records.Results[0].Series[0].Columns[i]] = val[i]
			}
			// append temp result to master result map
			resulto = append(resulto, tRes)
		}

		writeBody(resulto, res, http.StatusOK, req)
		return
	}

	result := point{}
	val := records.Results[0].Series[0].Values[0]
	result.Time, _ = val[0].(json.Number).Int64()
	result.Time = result.Time * 1000
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
		if key == ":stat" || key == "verb" || key == "start" || key == "stop" || key == "backfill" || key == "x-auth-token" || key == "X-AUTH-TOKEN" {
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

	bf := req.URL.Query().Get("backfill")

	if bf == "" {
		// grab the hourly averages ('group by' should aggregate multiple filters)
		query = fmt.Sprintf("%s GROUP BY time(1h) FILL(none)", query)
	} else {
		if s, err := strconv.ParseFloat(bf, 64); err != nil {
			// if bf is not a number, use 0 to backfill
			query = fmt.Sprintf("%s GROUP BY time(1h) FILL(0)", query)
		} else {
			// else use the number value of backfill
			query = fmt.Sprintf("%s GROUP BY time(1h) FILL(%v)", query, s)
		}
	}

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
		if key == ":stat" || key == "verb" || key == "start" || key == "stop" || key == "backfill" || key == "x-auth-token" || key == "X-AUTH-TOKEN" {
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
	query = fmt.Sprintf("%s GROUP BY time(15m) FILL(none)", query)

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
		id := fmt.Sprintf("%02d:%02d", hour, minute)
		valData, _ := values[1].(json.Number).Float64()

		// keep a running average instead of a total
		result[id] = ((result[id] * float64(counter[id])) + valData) / float64(counter[id]+1)
		counter[id] += 1
	}

	bf := req.URL.Query().Get("backfill")

	// if backfill set to anything, fill in the missing data with 0 value
	if bf != "" {
		daytimes := []string{"00:00", "00:15", "00:30", "00:45", "01:00", "01:15", "01:30", "01:45", "02:00", "02:15", "02:30", "02:45",
			"03:00", "03:15", "03:30", "03:45", "04:00", "04:15", "04:30", "04:45", "05:00", "05:15", "05:30", "05:45",
			"06:00", "06:15", "06:30", "06:45", "07:00", "07:15", "07:30", "07:45", "08:00", "08:15", "08:30", "08:45",
			"09:00", "09:15", "09:30", "09:45", "10:00", "10:15", "10:30", "10:45", "11:00", "11:15", "11:30", "11:45",
			"12:00", "12:15", "12:30", "12:45", "13:00", "13:15", "13:30", "13:45", "14:00", "14:15", "14:30", "14:45",
			"15:00", "15:15", "15:30", "15:45", "16:00", "16:15", "16:30", "16:45", "17:00", "17:15", "17:30", "17:45",
			"18:00", "18:15", "18:30", "18:45", "19:00", "19:15", "19:30", "19:45", "20:00", "20:15", "20:30", "20:45",
			"21:00", "21:15", "21:30", "21:45", "22:00", "22:15", "22:30", "22:45", "23:00", "23:15", "23:30", "23:45"}

		// set default backfill value
		bfFloat := 0.0

		// if what was passed in is a valid number, reset backfill to that
		if s, err := strconv.ParseFloat(bf, 64); err == nil {
			bfFloat = s
		}

		for i := range daytimes {
			if _, ok := result[daytimes[i]]; !ok {
				result[daytimes[i]] = bfFloat
			}
		}
	}

	// todo: will it be better&possible(backfill) to just use a struct slice the whole time?
	// convert result map to struct slice (make more uniform with hourly response)
	dailies := mapToStructSlice(result)

	writeBody(dailies, res, http.StatusOK, req)
}

// converts a map to a struct slice (uniformity)
func mapToStructSlice(mappy map[string]float64) []dayPoint {
	thing := []dayPoint{}

	for k, v := range mappy {
		thang := dayPoint{Time: k, Value: v}
		thing = append(thing, thang)
	}

	return thing
}
