package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
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

func statRequest(res http.ResponseWriter, req *http.Request) {
	rec, err := getStats(res, req)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	if len(rec.Series) != 1 {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	result := make([]point, len(rec.Series[0].Values))
	for idx, values := range rec.Series[0].Values {
		temp, _:= values[0].(json.Number).Int64()
		result[idx].Time     = temp * 1000
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
	rec, err := getStats(res, req)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	if len(rec.Series) != 1 {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	result := make(map[string]float64, 24*4)

	for _, values := range rec.Series[0].Values {
		valTime, _ := values[0].(json.Number).Int64()
		valUnix := time.Unix(valTime, 0)
		hour, minute, _ := valUnix.Clock()
		minute = int(math.Floor(float64(minute/15.0)) * 15)
		id := fmt.Sprintf("%v:%v", hour, minute)
		valData, _ := values[1].(json.Number).Float64()
		result[id] += valData
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}
	res.Write(append(bytes, byte('\n')))
}

func getStats(res http.ResponseWriter, req *http.Request) (*client.Result, error) {
	service := req.URL.Query().Get(":service")
	stat := req.URL.Query().Get(":stat")
	query := fmt.Sprintf(`select "%v" from "1.week".metrics where service = '%v'`, stat, service)
	records, err := influx.Query(query)
	if err != nil {
		return nil, err
	}
	return &records.Results[0], nil
}
