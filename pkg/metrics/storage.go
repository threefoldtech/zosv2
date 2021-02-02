package metrics

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

// Average is an average value over a specific period of time
type Average struct {
	Width     int64
	Timestamp int64
	Value     float64
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
	Update(name, id string, mode aggregated.AggregationMode, value float64) error
	Metrics(name string) ([]Metric, error)
}
