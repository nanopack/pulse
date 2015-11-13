package collector

type (
	DataSet func() map[string]float64
	set     struct {
		Collect

		set     DataSet
		current map[string]float64
	}
)

func NewSetCollector(stats DataSet) Collector {
	set := &set{
		set: stats,
	}
	set.collectValue()
	set.CollectFun = set.collectValue

	return set
}

func (set *set) Values() map[string]float64 {
	return set.current
}

func (set *set) Flush() {
	set.current = make(map[string]float64, 0)
}

func (set *set) collectValue() {
	set.current = set.set()
}
