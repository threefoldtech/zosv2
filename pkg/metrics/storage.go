package metrics

import "github.com/threefoldtech/zos/pkg/metrics/aggregated"

// Metric struct holds the ID and average values
// as configured
type Metric struct {
	ID     string
	Values []float64
}

// Storage interface
type Storage interface {
	Update(name, id string, mode aggregated.AggregationMode, value float64) error
	Metrics(name string) ([]Metric, error)
}

type CPU interface {
	Update(name, id string, mode aggregated.AggregationMode, value float64) error
	Metrics(name string) ([]Metric, error)
}
