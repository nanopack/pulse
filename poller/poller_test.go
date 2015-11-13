package poller

import (
	"sort"
	"testing"
	"time"
)

func TestPoller(test *testing.T) {
	results := [][]string{
		{"cpu", "disk", "ram"},
		{"cpu"},
		{"cpu", "ram"},
		{"cpu", "disk"},
		{"cpu", "ram"},
		{"cpu"},
	}
	count := 0
	poller := NewPoller(func(tags []string) {
		sort.Strings(tags)
		// this might very rarely cause this test to fail.
		for idx, tag := range results[time.Now().Unix()%6] {
			assert(test, tag == tags[idx], "incorrect order of tags %v", tags)
		}
		count++
	})
	defer poller.Close()

	client := poller.NewClient()
	defer client.Close()

	client.Poll("cpu", 1)
	client.Poll("ram", 2)
	client.Poll("disk", 3)
	time.Sleep(time.Second*3 + time.Second/2)
	assert(test, count == 3, "poll function was only called %v times", count)
}

func assert(test *testing.T, check bool, fmt string, args ...interface{}) {
	if !check {
		test.Logf(fmt, args...)
		test.FailNow()
	}
}
