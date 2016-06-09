// Package kapacitor provides the means for alerting if a stat exceeds a threshold.
package kapacitor

// todo: store configured alerts

import (
	"fmt"
	"strings"
	"time"

	"github.com/influxdata/kapacitor/client/v1"
	"github.com/jcelliott/lumber"
	"github.com/spf13/viper"
)

var (
	cli *client.Client
)

// Task is an object for registering kapacitor tasks
type Task struct {
	Id              string `json:"id"`
	Type            string `json:"type"`
	Database        string `json:"database"`
	RetentionPolicy string `json:"retention-policy"`
	Script          string `json:"script"`
	Status          string `json:"status"`
}

// Alert is an object to simplify creating tasks
type Alert struct {
	Tags      map[string]string `json:"tags,omitempty"`  // populates the WHERE
	Metric    string            `json:"metric"`          // the stat to track
	Level     string            `json:"level,omitempty"` // the alert level (info, warn, crit)
	Threshold int               `json:"threshold"`       // limit that alert is triggered
	Duration  string            `json:"duration"`        // how far back to average (5m)
	Post      string            `json:"post"`            // api to hit when alert is triggered
}

// Init initializes the client
func Init() error {
	var err error
	cli, err = client.New(client.Config{
		URL:                viper.GetString("kapacitor-address"),
		Timeout:            5 * time.Second,
		UserAgent:          "PulseClient",
		InsecureSkipVerify: true, // todo: maybe set back to `viper.GetBool("insecure")`
	})
	if err != nil {
		return fmt.Errorf("Failed to create new client! - %v", err)
	}
	_, _, err = cli.Ping()
	return err
}

// SetTask adds or updates a kapacitor task
func SetTask(task Task) error {
	var Type client.TaskType
	var Status client.TaskStatus
	DBRPs := make([]client.DBRP, 1)

	// convert type
	switch strings.ToUpper(task.Type) {
	case "BATCH":
		Type = client.BatchTask
	case "STREAM":
		Type = client.StreamTask
	default:
		return fmt.Errorf("Bad task type - '%v'", task.Type)
	}

	DBRPs[0].Database = task.Database
	DBRPs[0].RetentionPolicy = task.RetentionPolicy

	// convert status
	switch strings.ToUpper(task.Status) {
	case "DISABLED":
		Status = client.Disabled
	case "ENABLED":
		Status = client.Enabled
	case "":
		// default to disabled
		Status = client.Disabled
	default:
		return fmt.Errorf("Bad task status - '%v'", task.Status)
	}

	var err error
	l := cli.TaskLink(task.Id)
	t, _ := cli.Task(l, nil)
	if t.ID == "" {
		_, err = cli.CreateTask(client.CreateTaskOptions{
			ID:         task.Id,
			Type:       Type,
			DBRPs:      DBRPs,
			TICKscript: task.Script,
			Status:     Status,
		})
		lumber.Trace("Task Created")
	} else {
		_, err = cli.UpdateTask(
			l,
			client.UpdateTaskOptions{
				Type:       Type,
				DBRPs:      DBRPs,
				TICKscript: task.Script,
			},
		)
		lumber.Trace("Task Updated")
	}
	if err != nil {
		return fmt.Errorf("Failed to create task - %v", err)
	}

	return nil
}

// DeleteTask removes a kapacitor task
func DeleteTask(id string) error {
	err := cli.DeleteTask(cli.TaskLink(id))
	if err != nil {
		err = fmt.Errorf("Failed to delete task - %v", err)
	}

	return err
}

// GenBatchTick generates a simple batch type TICKscript
func GenBatchTick(stat, database, retentionPolicy, measurement string, where map[string]string, period, every string, alerts map[string]string, post string) string {
	query := genQuery(stat, database, retentionPolicy, measurement, genWhere(where), period, every)
	alert := genAlert(alerts, post)
	return fmt.Sprintf("batch%s\n%s", query, alert)
}

// generate the alert portion of the TICKscript
func genAlert(alerts map[string]string, post string) string {
	return fmt.Sprintf(`
	|alert()
%s
		.log('/tmp/%s')`, genLambda(alerts), post)
}

// generate the lambda portion of the TICKscript
func genLambda(alerts map[string]string) string {
	t := []string{}
	for k, v := range alerts {
		t = append(t, fmt.Sprintf("\t\t.%s(lambda: %s)", k, v))
	}
	return strings.Join(t, "")
}

// generate the where portion of the TICKscript
func genWhere(wheres map[string]string) string {
	w := []string{}
	for k, v := range wheres {
		w = append(w, fmt.Sprintf("\"%v\" = '%v'", k, v))
	}
	return strings.Join(w, " AND ")
}

// generate the query portion of the TICKscript
func genQuery(stat, database, retentionPolicy, measurement, wheres, period, every string) string {
	return fmt.Sprintf(`
	|query('''
		SELECT mean(%s) AS mean_%s
		FROM "%s"."%s"."%s"
		WHERE %s
	''')
		.period(%s)
		.every(%s)`,
		stat, stat, database, retentionPolicy, measurement, wheres, period, every)
}
