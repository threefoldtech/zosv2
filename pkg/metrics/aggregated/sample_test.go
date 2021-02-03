package aggregated

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAlignedAvg(t *testing.T) {
	require := require.New(t)
	on := time.Date(2020, time.January, 1, 0, 3, 0, 0, time.UTC)
	aligned := NewAlignedSample(time.Date(2020, time.January, 1, 0, 3, 0, 0, time.UTC), 5*time.Minute)
	require.NotNil(aligned)

	require.Equal(
		time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		aligned.Time().Unix(),
	)

	require.Equal(float64(0), aligned.Average())
	avg, err := aligned.Sample(on, 10)
	require.NoError(err)
	require.Equal(float64(10), avg)
	avg, err = aligned.Sample(on, 20)
	require.NoError(err)
	require.Equal(float64(15), avg)
	avg, err = aligned.Sample(on, 30)
	require.NoError(err)
	require.Equal(float64(20), avg)

	_, err = aligned.Sample(time.Now(), 30)
	require.Equal(ErrValueIsAfterPeriod, err)
	_, err = aligned.Sample(time.Date(2019, time.December, 20, 0, 0, 0, 0, time.UTC), 30)
	require.Equal(ErrValueIsBeforePeriod, err)
}
