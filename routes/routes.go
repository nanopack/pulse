package routes

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"

	"github.com/nanobox-io/nanobox-api"
	"github.com/nanopack/pulse/server"
)

type (
	point struct {
		Time  int64   `json:"time"`
		Value float64 `json:"value"`
	}
)

func Init() {
	api.Router.Get("/services/{service}/stats/{stat}/hourly", api.TraceRequest(statRequest))
	api.Router.Get("/services/{service}/stats/{stat}/daily_peaks", api.TraceRequest(combinedRequest))
}

func statRequest(res http.ResponseWriter, req *http.Request) {
	rec, err := getStats(res, req)
	if err != nil {
		// do something here
	}
	if len(rec.Series) != 1 {
		// do some error stuff here
		return
	}
	result := make([]point, len(rec.Series[0].Values))
	for idx, values := range rec.Series[0].Values {
		result[idx].Time = values[0].(time.Time).Unix() * 1000
		result[idx].Value = values[1].(float64)
	}
	bytes, err := json.Marshal(result)
	if err != nil {

	}
	res.Write(bytes)
}

func combinedRequest(res http.ResponseWriter, req *http.Request) {
	rec, err := getStats(res, req)
	if err != nil {
		// do something here
	}
	if len(rec.Series) != 1 {
		// do some error stuff here
		return
	}
	// 15 minute intervals in one day.
	result := make(map[string]float64, 24*4)

	for _, values := range rec.Series[0].Values {
		hour, minute, _ := values[0].(time.Time).Clock()
		minute = int(math.Floor(float64(minute/15.0)) * 15)
		id := fmt.Sprintf("%v:%v", hour, minute)
		result[id] += values[1].(float64)
	}
	bytes, err := json.Marshal(result)
	if err != nil {

	}
	res.Write(bytes)
}

func getStats(res http.ResponseWriter, req *http.Request) (influx.Result, error) {
	server := api.User.(*server.Server)
	service := req.URL.Query().Get(":service")
	stat := req.URL.Query().Get(":stat")
	query := fmt.Sprintf(`select "%v" from "1.week".metrics where service = '%v'`, stat, service)
	fmt.Println(query)
	records, err := server.Query(query)
	if err != nil {
		return records.Results[0], err
	}
	return records.Results[0], nil
}
