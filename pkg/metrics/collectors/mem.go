package collectors

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

type memCollector struct {
	m    metrics.Storage
	keys []Metric
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector(storage metrics.Storage) Collector {
	return &memCollector{
		m: storage,
		keys: []Metric{
			{"utilization.mem.size", "average total memory size in bytes"},
			{"utilization.mem.free", "average free memory size in bytes"},
			{"utilization.mem.used", "average used memory size in bytes"},
			{"utilization.mem.available", "average available memory size in bytes"},
			{"utilization.mem.percent", "average memory usage percentage"},
		},
	}
}

func (m *memCollector) Metrics() []Metric {
	return m.keys
}

func (m *memCollector) Collect() error {
	stats, err := mem.VirtualMemory()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve memory statistics")
	}

	m.update("utilization.mem.size", float64(stats.Total))
	m.update("utilization.mem.free", float64(stats.Free))
	m.update("utilization.mem.used", float64(stats.Used))
	m.update("utilization.mem.available", float64(stats.Available))
	m.update("utilization.mem.percent", stats.UsedPercent)

	return nil
}

func (m *memCollector) update(key string, value float64) {
	if err := m.m.Update(key, "mem", aggregated.AverageMode, value); err != nil {
		log.Error().Err(err).Str("metric", key).Msg("failed to update metric")
	}
}
