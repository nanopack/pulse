package kapacitor_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/viper"

	"github.com/nanopack/pulse/kapacitor"
)

var (
	stat            = "cpu_used"
	database        = "test"
	retentionPolicy = "default"

	measurement = "cpu_used"
	where       = map[string]string{"region": "home", "host": ""}
	period      = "3m"
	every       = "30s"
	threshold   = 0
	post        = "alerts.log"

	alerts = map[string]string{"info": "\"mean_cpu_used\" > 0"}
)

func TestMain(m *testing.M) {
	// viper.SetDefault("kapacitor-address", "http://localhost:9092")
	viper.SetDefault("kapacitor-address", "http://172.28.128.4:9092")
	err := kapacitor.Init()
	if err != nil {
		fmt.Printf("Failed to init - '%v' skipping tests\n", err)
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func TestCreateUpdateTask(t *testing.T) {
	task := kapacitor.Task{
		Id:              stat,
		Type:            "batch",
		Database:        database,
		RetentionPolicy: retentionPolicy,
		Script:          kapacitor.GenBatchTick(stat, database, retentionPolicy, measurement, where, period, every, alerts, post),
		Status:          "enabled",
	}
	err := kapacitor.SetTask(task)
	if err != nil {
		t.Errorf("Failed to add task - %v", err)
	}

	// to make work with repo update from 4767348... TO 3003b83...
	// requires using kapacitord 1.0.0-beta or nightlies.
	task.Status = "disabled"
	err = kapacitor.SetTask(task)
	if err != nil {
		t.Errorf("Failed to update task - %v", err)
	}

	task.Status = ""
	err = kapacitor.SetTask(task)
	if err != nil {
		t.Errorf("Failed to test task status blank - %v", err)
	}

	task.Status = "bad"
	err = kapacitor.SetTask(task)
	if err == nil {
		t.Error("Failed to fail bad task status")
	}

	task.Type = "stream"
	err = kapacitor.SetTask(task)
	if err == nil {
		t.Errorf("Failed to fail add stream task - %v", err)
	}

	task.Type = "bad"
	err = kapacitor.SetTask(task)
	if err == nil {
		t.Errorf("Failed to fail add bad task type - %v", err)
	}
}

func TestDeleteTask(t *testing.T) {
	err := kapacitor.DeleteTask(stat)
	if err != nil {
		t.Errorf("Failed to delete task - %v", err)
	}
}
