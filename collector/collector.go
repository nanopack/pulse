package collector

import (
	"time"
)

type (
	Collector interface {
		Stop()
		Start()
		Values() map[string]float64
		Flush()
		SetInterval(time.Duration)
		OverrideInterval(time.Duration, time.Duration)
	}

	Collect struct {
		done       chan interface{}
		next       <-chan time.Time
		revert     chan bool
		interval   time.Duration
		override   time.Duration
		CollectFun func()
	}
)

func (collector *Collect) SetInterval(interval time.Duration) {
	collector.interval = interval
	collector.reset()
}

func (collector *Collect) OverrideInterval(newInterval time.Duration, howLong time.Duration) {
	if collector.override != 0 {
		close(collector.revert)
	}
	collector.override = newInterval
	collector.revert = make(chan bool)
	collector.reset()
	go func() {
		select {
		case <-time.After(howLong):
			collector.override = 0
			collector.reset()
		case <-collector.revert:
			return
		}
	}()
}

func (collector *Collect) reset() {
	switch {
	case collector.override != 0:
		collector.next = time.After(collector.override)
	default:
		collector.next = time.After(collector.interval)
	}

}

func (collector *Collect) Stop() {
	if collector.done != nil {
		close(collector.done)
		collector.done = nil
	}
}

func (collector *Collect) Start() {
	if collector.done == nil {
		collector.reset()
		collector.done = make(chan interface{})
		go func() {
			for {
				select {
				case <-collector.done:
					return
				case <-collector.next:
					collector.reset()
					collector.CollectFun()
				}
			}
		}()
	}
}
