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
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusBadRequest)
		return
	}

	// verify we have enough info
	if alert.Metric == "" || alert.Post == "" {
		writeBody(apiError{ErrorString: "Missing value in payload"}, res, http.StatusBadRequest)
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
		Id:              alert.Metric,
		Type:            "batch",
		Database:        "statistics",
		RetentionPolicy: "two_days",
		Script:          kapacitor.GenBatchTick(alert.Metric, "statistics", "two_days", alert.Metric, alert.Tags, alert.Duration, "30s", lambda, alert.Post),
		Status:          "enabled",
	}

	err = kapacitor.SetTask(task)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusBadRequest)
		return
	}

	writeBody(alert, res, http.StatusOK)
}

// delete a task
func deleteAlert(res http.ResponseWriter, req *http.Request) {
	taskId := req.URL.Query().Get(":id")

	err := kapacitor.DeleteTask(taskId)
	if err != nil {
		writeBody(apiError{ErrorString: err.Error()}, res, http.StatusBadRequest)
		return
	}

	writeBody(apiMsg{"Success"}, res, http.StatusOK)
}
