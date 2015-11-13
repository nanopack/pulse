package plexer

import (
	"testing"
	"time"
)

func TestPlex(test *testing.T) {
	plex := NewPlexer()
	count := 0
	plex.AddObserver("test", func(tags []string, data string) error {
		count++
		return nil
	})

	plex.PublishSingle([]string{"cpu_used"}, "6")
	plex.PublishSingle([]string{"cpu_used"}, "6")
	time.Sleep(time.Millisecond * 10)
	assert(test, count == 2, "publisher was called an incorrect number of times")
	plex.RemoveObserver("test")
	plex.PublishSingle([]string{"cpu_used"}, "6")
	time.Sleep(time.Millisecond * 10)
	assert(test, count == 2, "publisher was called an incorrect number of times")
}

func assert(test *testing.T, check bool, fmt string, args ...interface{}) {
	if !check {
		test.Logf(fmt, args...)
		test.FailNow()
	}
}
