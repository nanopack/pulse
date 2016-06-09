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

	task := kapacitor.Task{
		// todo: make id more unique, for use with same stat, multiple tags
		Id:              alert.Metric,
		Type:            "batch",
		Database:        "statistics",
		RetentionPolicy: "one_day",
		Script:          kapacitor.GenBatchTick(alert.Metric, "statistics", "one_day", alert.Metric, alert.Tags, alert.Duration, "30s", lambda, alert.Post),
		Status:          "enabled",
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
	// BUG(glinton) ids are not unique enough for the same stat to be used with different tag sets
	// todo: make id more unique, for use with same stat, multiple tags
	taskId := req.URL.Query().Get(":id")

	err := kapacitor.DeleteTask(taskId)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusBadRequest, req)
		return
	}

	writeBody(apiMsg{"Success"}, res, http.StatusOK, req)
}
