package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client/v2"

	"github.com/nanopack/pulse/influx"
)

type (
	point struct {
		Time  int64   `json:"time"`
		Value float64 `json:"value"`
	}
)

func keysRequest(res http.ResponseWriter, req *http.Request) {
	cols, err := influx.Query("SHOW FIELD KEYS FROM two_days.aggregate") // will only be populated after data is aggregated ~15m
	if err != nil {
		panic(err)
	}

	columns := []string{}
	for _, result := range cols.Results {
		for _, series := range result.Series {
			for _, val := range series.Values {
				columns = append(columns, val[0].(string))
			}
		}
	}

	writeBody(columns, res, http.StatusOK)
}

func tagsRequest(res http.ResponseWriter, req *http.Request) {
	// check tags
	groupBy, err := influx.Query("SHOW TAG KEYS FROM two_days.aggregate")
	if err != nil {
		panic(err)
	}

	group := []string{}
	for _, result := range groupBy.Results {
		for _, series := range result.Series {
			for _, val := range series.Values {
				group = append(group, val[0].(string))
			}
		}
	}

	writeBody(group, res, http.StatusOK)
}

func statRequest(res http.ResponseWriter, req *http.Request) {
	rec, err := getStats(req)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	if len(rec.Series) != 1 {
		res.WriteHeader(404)
		return
	}
	result := make([]point, len(rec.Series[0].Values))
	for idx, values := range rec.Series[0].Values {
		temp, _ := values[0].(json.Number).Int64()
		result[idx].Time = temp * 1000
		result[idx].Value, _ = values[1].(json.Number).Float64()
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	res.Write(append(bytes, byte('\n')))
}

func combinedRequest(res http.ResponseWriter, req *http.Request) {
	rec, err := getStats(req)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	if len(rec.Series) != 1 {
		res.WriteHeader(404)
		return
	}
	result := make(map[string]float64, 24*4)
	counter := make(map[string]int, 24*4)

	for _, values := range rec.Series[0].Values {
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

	bytes, err := json.Marshal(result)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	res.Write(append(bytes, byte('\n')))
}

func getStats(req *http.Request) (*client.Result, error) {
	stat := req.URL.Query().Get(":stat")
	query := fmt.Sprintf(`select "%v" from one_week.aggregate`, stat) // same as one_week./.*/
	filters := []string{}
	for key, val := range req.URL.Query() {
		if key == ":stat" {
			continue
		}
		filters = append(filters, fmt.Sprintf("%s = '%s'", key, val[0]))
	}
	fmt.Println(filters)
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}
	records, err := influx.Query(query)
	if err != nil {
		return nil, err
	}
	return &records.Results[0], nil
}
