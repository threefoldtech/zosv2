package aggregated

import (
	"sync"
	"time"
)

// AggregationMode type
type AggregationMode int

const (
	// AverageMode aggregation mode always keep track of the average over
	// the configured periods
	AverageMode AggregationMode = iota
	// DifferentialMode aggregation mode always keeps track of the difference from
	// the last value
	DifferentialMode
)

// Aggregated represents an aggregated value
type Aggregated struct {
	Mode      AggregationMode `json:"mode"`
	Samples   []*Sample       `json:"samples"`
	Durations []time.Duration `json:"durations"`
	Last      float64         `json:"last"`
	m         sync.RWMutex    `json:"_"`
}

// NewAggregatedMetric returns an aggregated metric over given period
func NewAggregatedMetric(mode AggregationMode, durations ...time.Duration) Aggregated {
	if len(durations) == 0 {
		panic("at least one duration is needed")
	}

	return Aggregated{Mode: mode, Durations: durations}
}

func (a *Aggregated) sample(t time.Time, value float64) {
	a.m.Lock()
	defer a.m.Unlock()

	if len(a.Samples) == 0 {
		for _, d := range a.Durations {
			a.Samples = append(a.Samples, NewAlignedSample(t, d))
		}
	}

	last := a.Last
	a.Last = value
	if a.Mode == DifferentialMode {
		// probably first update, so we keep track
		// only of last value.
		if last == 0 {
			return
		}
		// otherwise the value is the difference (increase)
		value = last - value
	}

	// update all samples
	for i, s := range a.Samples {
		err := s.Sample(t, value)
		if err == ErrValueIsAfterPeriod {
			// sample period has passed, so we need to
			// create a new sample.
			// QUESTION: push this sample to history?
			s = NewAlignedSample(t, s.Width())
			s.Sample(t, value)
			a.Samples[i] = s
		}
	}
}

// Sample update the aggregated value with given values
func (a *Aggregated) Sample(value float64) {
	a.sample(time.Now(), value)
}

// Averages return the averages per configured duration
func (a *Aggregated) Averages() []float64 {
	a.m.RLock()
	defer a.m.RUnlock()

	v := make([]float64, len(a.Durations))
	for i, sample := range a.Samples {
		v[i] = sample.Average()
	}

	return v
}
