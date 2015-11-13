package collector

import (
	"math"
)

func MutedAverage(point DataPoint, amount float64) DataPoint {
	current := float64(0)
	return func() float64 {
		current += point() / amount
		return current
	}
}

func Average(point DataPoint) DataPoint {
	count := float64(0)
	current := float64(0)
	return func() float64 {
		// has to be a better way to do this
		current = (current*count + point()) / (count + 1)
		count++
		return current
	}
}

func RunningAverage(point DataPoint, length int) DataPoint {
	values := make([]float64, length)
	idx := 0
	return func() float64 {
		values[idx%length] = point()
		idx++
		count := float64(0)
		value := float64(0)
		for _, val := range values {
			value += val
			count++
		}
		count = float64(math.Min(count, float64(idx)))
		if count == 0 {
			return 0
		}
		return value / count
	}
}
