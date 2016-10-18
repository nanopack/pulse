package api

import (
	"fmt"
	"net/http"

	"github.com/nanopack/pulse/kapacitor"
)

// add or update a kapacitor task
func setAlert(res http.ResponseWriter, req *http.Request) {
	var alert kapacitor.Alert
	err := parseBody(req, &alert)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusBadRequest, req)
		return
	}

	// verify we have enough info
	if alert.Metric == "" || alert.Post == "" {
		writeBody(apiError{ErrorString: "Missing value in payload"}, res, http.StatusBadRequest, req)
		return
	}

	// set sane defaults
	if alert.Level == "" {
		alert.Level = "crit"
	}
	if alert.Duration == "" {
		alert.Duration = "5m"
	}

	lambda := map[string]string{alert.Level: fmt.Sprintf("\"mean_%s\" > %d", alert.Metric, alert.Threshold)}

	// generate id to set for task/return to user
	alert.GenId()

	task := kapacitor.Task{
		Id:              alert.Id,
		Type:            "batch",
		Database:        "statistics",
		RetentionPolicy: "one_day",
		Status:          "enabled",
		Script:          kapacitor.GenBatchTick(alert.Metric, "statistics", "one_day", alert.Metric, alert.Tags, alert.Duration, "30s", lambda, alert.Post),
		// Script:       GenBatchTick(stat, database, retentionPolicy, measurement string, where map[string]string, period, every string, alerts map[string]string, post string) string,
	}

	err = kapacitor.SetTask(task)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusBadRequest, req)
		return
	}

	writeBody(alert, res, http.StatusOK, req)
}

// delete a task
func deleteAlert(res http.ResponseWriter, req *http.Request) {
	taskId := req.URL.Query().Get(":id")

	err := kapacitor.DeleteTask(taskId)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusBadRequest, req)
		return
	}

	writeBody(apiMsg{"Success"}, res, http.StatusOK, req)
}
