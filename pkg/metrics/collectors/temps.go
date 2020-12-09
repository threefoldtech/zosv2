package collectors

import (
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

// CPUCollector type
type tempsCollector struct {
	m metrics.Sensor

	keys []Metric
}

// NewTempsCollector created a disk collector
func NewTempsCollector(storage metrics.Storage) Collector {
	return &tempsCollector{
		m: storage,
		keys: []Metric{
			{"node.sensor.reading", "average value reported by sensor"},
		},
	}
}

func (d *tempsCollector) collectSensors() error {
	sensors, err := host.SensorsTemperatures()
	if err != nil {
		return errors.Wrap(err, "failed to get temperature stats")
	}

	for _, tempStat := range sensors {
		d.m.Update("node.sensor.reading", tempStat.SensorKey, aggregated.AverageMode, tempStat.Temperature)
	}

	return nil
}

func (d *tempsCollector) Metrics() []Metric {
	return d.keys
}

// Collect method
func (d *tempsCollector) Collect() error {
	return d.collectSensors()
}
