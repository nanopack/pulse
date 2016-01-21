package relay

import (
)

type (
	Collector interface {
		Collect() map[string]float64
	}

	collectorHandle func() map[string]float64
)

func (c collectorHandle) Collect() map[string]float64 {
	return c()
}

func NewPointCollector(pf func() float64) Collector {
	return collectorHandle(func() map[string]float64 {
		return map[string]float64{"": pf()}
	})
}

func NewSetCollector(sf collectorHandle) Collector {
	return sf
}

