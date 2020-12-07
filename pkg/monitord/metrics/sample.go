package metrics

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

// Sample interface
type Sample interface {
	Sample(time.Time, float64) error
	Last() float64
	Average() float64
	Max() float64
	Time() time.Time
}

type alignedSample struct {
	at    int64
	width int64
	last  float64
	total float64
	count float64
	max   float64
}

//NewAlignedSample aligned sample makes sure sample is always aligned to given duration
func NewAlignedSample(at time.Time, width time.Duration) Sample {
	// time need to be aligned per
	widthSeconds := int64(width / time.Second)

	return &alignedSample{
		at:    (at.Unix() / widthSeconds) * widthSeconds,
		width: widthSeconds,
	}
}

func (s *alignedSample) Time() time.Time {
	return time.Unix(s.at, 0)
}

func (s *alignedSample) Sample(t time.Time, v float64) error {
	aligned := (t.Unix() / s.width) * s.width
	if aligned > s.at {
		// this sample happened after this period is over
		return ErrValueIsAfterPeriod
	} else if aligned < s.at {
		// this sample happened before the period has started
		return ErrValueIsBeforePeriod
	}

	if v > s.max {
		s.max = v
	}

	s.count++
	s.total += v
	s.last = v
	return nil
}

func (s *alignedSample) Max() float64 {
	return s.max
}

func (s *alignedSample) Last() float64 {
	return s.last
}

func (s *alignedSample) Average() float64 {
	if s.count != 0 {
		return s.total / s.count
	}
	return 0
}
