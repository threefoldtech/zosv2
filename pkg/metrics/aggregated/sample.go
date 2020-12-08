package aggregated

import (
	"fmt"
	"time"
)

var (
	//ErrValueIsAfterPeriod is returned when the given value is outside (after) the period
	ErrValueIsAfterPeriod = fmt.Errorf("provided sample is after defined period")
	//ErrValueIsBeforePeriod is returned when the given value is outside (before) the period
	ErrValueIsBeforePeriod = fmt.Errorf("provided sample is before defined period")
)

// Sample is an averaged value over the defined period.
type Sample struct {
	// Timestamp of this sample
	Timestamp int64 `json:"timestamp"`
	// Duration in seconds
	Duration int64 `json:"duration"`
	// Last feeded value
	Last float64 `json:"last"`
	// Total (sum) of all feeded values
	Total float64 `json:"total"`
	// Count number of samples
	Count float64 `json:"count"`
	// Max feeded value
	Max float64 `json:"max"`
}

//NewAlignedSample aligned sample makes sure sample is always aligned to given duration
func NewAlignedSample(at time.Time, width time.Duration) *Sample {
	// time need to be aligned per
	widthSeconds := int64(width / time.Second)

	return &Sample{
		Timestamp: (at.Unix() / widthSeconds) * widthSeconds,
		Duration:  widthSeconds,
	}
}

// Time gets timestamp as time.Time object
func (s *Sample) Time() time.Time {
	return time.Unix(s.Timestamp, 0)
}

// Width gets duration as time.Duration
func (s *Sample) Width() time.Duration {
	return time.Duration(s.Duration) * time.Second
}

// Sample update this sample with given value at given time
func (s *Sample) Sample(t time.Time, v float64) error {
	aligned := (t.Unix() / s.Duration) * s.Duration
	if aligned > s.Timestamp {
		// this sample happened after this period is over
		return ErrValueIsAfterPeriod
	} else if aligned < s.Timestamp {
		// this sample happened before the period has started
		return ErrValueIsBeforePeriod
	}

	if v > s.Max {
		s.Max = v
	}

	s.Count++
	s.Total += v
	s.Last = v
	return nil
}

// Average returns the average value
func (s *Sample) Average() float64 {
	if s.Count != 0 {
		return s.Total / s.Count
	}
	return 0
}

func (s *Sample) String() string {
	return fmt.Sprintf("sample(%d, %d, %f)", s.Timestamp, s.Duration, s.Average())
}
