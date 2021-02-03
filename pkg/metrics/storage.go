package metrics

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

// Average is an average value over a specific period of time
type Average struct {
	// Width of the period in seconds
	Width int64
	// Timestamp of the period
	Timestamp int64
	// Value is the average value over the period [timestamp, timestamp+width]
	Value float64
}

func (a *Average) String() string {
	return fmt.Sprintf("(%d, %d, %f)", a.Width, a.Timestamp, a.Value)
}

// Metric struct holds the ID and average values
// as configured
type Metric struct {
	ID     string
	Values []Average
}

// Storage interface
type Storage interface {
	// Update updates the internal value associated with this metric key, it then
	// returns the calculated average for the current period.
	// the period itself is defined by the underlying implementations.
	Update(name, id string, mode aggregated.AggregationMode, value float64) (float64, error)
	// Metrics returns the values of the given metric name
	Metrics(name string) ([]Metric, error)
}
