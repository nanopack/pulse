package collector

type (
	DataPoint func() float64
	gauge     struct {
		Collect

		stat    DataPoint
		current float64
	}
)

func NewPointCollector(stat DataPoint) Collector {
	gauge := &gauge{
		stat: stat,
	}
	gauge.collectValue()
	gauge.CollectFun = gauge.collectValue

	return gauge
}

func (gauge *gauge) Values() map[string]float64 {
	return map[string]float64{"": gauge.current}
}

func (gauge *gauge) Flush() {
	gauge.current = 0
}

func (gauge *gauge) collectValue() {
	gauge.current = gauge.stat()
}
