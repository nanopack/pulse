package influx_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/nanopack/pulse/influx"
	"github.com/nanopack/pulse/plexer"
)

func TestInflux(t *testing.T) {
	// start influx

	// configure influx to connect to (DO NOT TEST ON PRODUCTION)
	viper.SetDefault("influx_address", "http://localhost:8086")
	viper.SetDefault("aggregate_interval", 1)

	// start cq checker
	go influx.KeepContinuousQueriesUpToDate()

	// initialize influx
	queries := []string{
		// clean influx to test with (DO NOT RUN ON PRODUCTION)
		"DROP   DATABASE statistics",
		"CREATE DATABASE statistics",
		`CREATE RETENTION POLICY "2.days" ON statistics DURATION 2d REPLICATION 1 DEFAULT`,
		`CREATE RETENTION POLICY "1.week" ON statistics DURATION 1w REPLICATION 1`,
	}
	for _, query := range queries {
		_, err := influx.Query(query)
		if err != nil {
			t.Error("Failed to QUERY/INITIALIZE - ", err)
		}
	}
}
func TestInsert(t *testing.T) {
	// define fake messages
	msg1 := plexer.Message{Tags: []string{"cpu_used", "cpu_not_free"}, Data: "0.34"}
	msg2 := plexer.Message{Tags: []string{"ram_used", "ram_not_free"}, Data: "0.43"}
	messages := plexer.MessageSet{Tags: []string{"host:tester", "test0"}, Messages: []plexer.Message{msg1, msg2}}

	// test inserting into influx
	if err := influx.Insert(messages); err != nil {
		t.Error("Failed to INSERT messages - ", err)
	}
}

func TestQuery(t *testing.T) {
	// ensure insert worked
	response, err := influx.Query(`Select * from "2.days".metrics`)
	if err != nil {
		t.Error("Failed to QUERY influx - ", err)
	}

	cpu_used := response.Results[0].Series[0].Values[0][1]

	if cpu_used != json.Number("0.34") {
		t.Error("Failed to QUERY influx - ( BAD INSERT: expected: 0.34, got: ", cpu_used, ")")
	}
	t.Logf("Waiting 65s for query to update")
}

func TestContinuousQuery(t *testing.T) {
	// wait for it to update
	time.Sleep(time.Second * 65)

	// ensure insert worked
	response, err := influx.Query(`SHOW CONTINUOUS QUERIES`)
	if err != nil {
		t.Error("Failed to QUERY influx - ", err)
	}

	cq := response.Results[0].Series[1].Values[0][1]
	if cq != `CREATE CONTINUOUS QUERY aggregate ON statistics BEGIN SELECT mean(cpu_used) AS "cpu_used", mean(ram_used) AS "ram_used" INTO statistics."1.week".metrics FROM statistics."2.days".metrics GROUP BY time(1m), host END` {
		t.Error("Failed to UPDATE CONTINUOUS QUERY influx")
	}
}
