package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/nanopack/pulse/api"
	"github.com/nanopack/pulse/kapacitor"
)

var apiAddr = "0.0.0.0:9898"

func TestMain(m *testing.M) {
	viper.SetDefault("http-listen-address", apiAddr)
	// viper.SetDefault("influx-address", "http://192.168.0.50:8086")
	// viper.SetDefault("kapacitor-address", "http://192.168.0.50:9092")
	viper.SetDefault("kapacitor-address", "http://127.0.0.1:9092")
	viper.SetDefault("influx-address", "http://127.0.0.1:8086")
	viper.SetDefault("insecure", true)
	viper.SetDefault("log-level", "trace")

	// start api
	go api.Start()
	kapacitor.Init()

	time.Sleep(time.Second)

	rtn := m.Run()

	os.Exit(rtn)
}

// test ping
func TestPing(t *testing.T) {
	resp, err := rest("GET", "/ping", "")
	if err != nil {
		t.Error(err)
	}

	if string(resp) != "pong" {
		t.Errorf("%s doesn't match expected out", resp)
	}
}

func TestGetKeys(t *testing.T) {
	resp, err := rest("GET", "/keys", "")
	if err != nil {
		t.Error(err)
	}

	if string(resp) != "null\n" {
		t.Errorf("%s doesn't match expected out", resp)
	}
}

func TestGetTags(t *testing.T) {
	resp, err := rest("GET", "/tags", "")
	if err != nil {
		t.Error(err)
	}

	if string(resp) != "null\n" {
		t.Errorf("%s doesn't match expected out", resp)
	}
}

func TestGetLatest(t *testing.T) {
	resp, err := rest("GET", "/latest/test-stat", "")
	if err != nil {
		t.Error(err)
	}

	if string(resp) != "{\"error\":\"Not Found\"}\n" {
		t.Errorf("%s doesn't match expected out", resp)
	}
}

func TestGetHourly(t *testing.T) {
	resp, err := rest("GET", "/hourly/test-stat", "")
	if err != nil {
		t.Error(err)
	}

	if string(resp) != "{\"error\":\"Not Found\"}\n" {
		t.Errorf("%s doesn't match expected out", resp)
	}
}

func TestGetDaily(t *testing.T) {
	resp, err := rest("GET", "/daily/test-stat", "")
	if err != nil {
		t.Error(err)
	}

	if string(resp) != "{\"error\":\"Not Found\"}\n" {
		t.Errorf("%s doesn't match expected out", resp)
	}

	resp, err = rest("GET", "/daily_peak/test-stat", "")
	if err != nil {
		t.Error(err)
	}

	if string(resp) != "404 page not found\n" {
		t.Errorf("%s doesn't match expected out", resp)
	}
}

func TestAddAlert(t *testing.T) {
	alert := `{
	  "tags": {"host":"abcd"},
	  "metric": "cpu_used",
	  "level": "crit",
	  "threshold": 80,
	  "duration": "30s",
	  "post": "http://127.0.0.1/alert"
	}`
	resp, err := rest("POST", "/alerts", alert)
	if err != nil {
		t.Error(err)
	}

	var newAlert kapacitor.Alert
	err = json.Unmarshal(resp, &newAlert)
	if err != nil {
		t.Errorf("Failed to POST alert - %s", err)
		t.FailNow()
	}

	if newAlert.Metric != "cpu_used" {
		t.Errorf("'%s' doesn't match expected out", newAlert.Metric)
	}

	resp, err = rest("DELETE", "/alerts/"+newAlert.Id, "")
	if err != nil {
		t.Error(err)
	}

	resp, err = rest("PUT", "/alerts", alert)
	if err != nil {
		t.Error(err)
	}

	err = json.Unmarshal(resp, &newAlert)
	if err != nil {
		t.Errorf("Failed to POST alert - %s", err)
		t.FailNow()
	}

	if newAlert.Metric != "cpu_used" {
		t.Errorf("'%s' doesn't match expected out", newAlert.Metric)
	}

	resp, err = rest("DELETE", "/alerts/"+newAlert.Id, "")
	if err != nil {
		t.Error(err)
	}
}

// hit api and return response body
func rest(method, route, data string) ([]byte, error) {
	body := bytes.NewBuffer([]byte(data))

	req, _ := http.NewRequest(method, fmt.Sprintf("http://%s%s", apiAddr, route), body)
	req.Header.Add("X-AUTH-TOKEN", "")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Unable to %s %s - %s", method, route, err)
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)

	return b, nil
}
