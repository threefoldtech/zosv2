package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAggregatedDurationInitial(t *testing.T) {
	require := require.New(t)
	value := NewAggregatedMetric(Average, time.Minute, 5*time.Minute)

	averages := value.Averages()
	require.Len(averages, 2)
	require.Equal([]float64{0, 0}, averages)
}

func TestAggregatedDurationAvg(t *testing.T) {
	require := require.New(t)
	aggregated := NewAggregatedMetric(Average, time.Minute, 5*time.Minute)

	start := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local)
	// we will feed values as follows
	// 1st  min [ 2 3 4 ] 2nd min [ 5 6 7 ] 3rd min
	// we check the averaged values
	// after feeding first minute, we should have same avg over the 1 and 5 min
	// on feeding 2nd min the values will differ
	// type input struct {
	// 	value float64
	// 	offset time.Duration
	// }
	type input struct {
		offset time.Duration
		value  float64
	}

	tests := []struct {
		inputs   []input
		expected []float64
	}{
		// first burst of values (fill first 1 min)
		{
			inputs:   []input{{10 * time.Second, 2}, {20 * time.Second, 3}, {40 * time.Second, 4}},
			expected: []float64{float64(2+3+4) / 3, float64(2+3+4) / 3},
		},
		// second burst of values (fill 2nd min)
		{
			inputs:   []input{{70 * time.Second, 5}, {80 * time.Second, 6}, {90 * time.Second, 7}},
			expected: []float64{float64(5+6+7) / 3, float64(2+3+4+5+6+7) / 6},
		},
		// 3rd
		{
			inputs:   []input{{130 * time.Second, 1}, {140 * time.Second, 2}, {150 * time.Second, 3}},
			expected: []float64{float64(1+2+3) / 3, float64(2+3+4+5+6+7+1+2+3) / 9},
		},
	}

	for _, testSample := range tests {
		for _, in := range testSample.inputs {
			aggregated.sample(start.Add(in.offset), in.value)
		}

		require.Equal(testSample.expected, aggregated.Averages())
	}
}

func TestAggregatedDurationDif(t *testing.T) {
	require := require.New(t)
	aggregated := NewAggregatedMetric(Differential, time.Minute, 5*time.Minute)

	start := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local)
	// we will feed values as follows
	// 1st  min [ 2 3 4 ] 2nd min [ 5 6 7 ] 3rd min
	// we check the averaged values
	// after feeding first minute, we should have same avg over the 1 and 5 min
	// on feeding 2nd min the values will differ
	// type input struct {
	// 	value float64
	// 	offset time.Duration
	// }
	type input struct {
		offset time.Duration
		value  float64
	}

	tests := []struct {
		inputs   []input
		expected []float64
	}{
		// first burst of values (fill first 1 min)
		{
			inputs:   []input{{10 * time.Second, 2}, {20 * time.Second, 3}, {40 * time.Second, 4}},
			expected: []float64{float64(1+1+1) / 3, float64(1+1+1) / 3},
		},
		// second burst of values (fill 2nd min)
		{
			inputs:   []input{{70 * time.Second, 5}, {80 * time.Second, 6}, {90 * time.Second, 7}},
			expected: []float64{float64(1+1+1) / 3, float64(1+1+1+1+1+1) / 6},
		},
		// 3rd
		{
			inputs:   []input{{130 * time.Second, 8}, {140 * time.Second, 9}, {150 * time.Second, 10}},
			expected: []float64{float64(1+1+1) / 3, float64(1+1+1+1+1+1+1+1+1) / 9},
		},
	}

	for _, testSample := range tests {
		for _, in := range testSample.inputs {
			aggregated.sample(start.Add(in.offset), in.value)
		}

		require.Equal(testSample.expected, aggregated.Averages())
	}
}
