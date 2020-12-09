package collectors

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

type memCollector struct {
	m    metrics.Storage
	keys []string
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector(storage metrics.Storage) Collector {
	return &memCollector{
		m: storage,
		keys: []string{
			"node.mem.size",
			"node.mem.free",
			"node.mem.used",
			"node.mem.available",
			"node.mem.percent",
		},
	}
}

func (m *memCollector) Metrics() []string {
	return m.keys
}

func (m *memCollector) Collect() error {
	stats, err := mem.VirtualMemory()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve memory statistics")
	}

	m.update("node.mem.size", float64(stats.Total))
	m.update("node.mem.free", float64(stats.Free))
	m.update("node.mem.used", float64(stats.Used))
	m.update("node.mem.available", float64(stats.Available))
	m.update("node.mem.percent", stats.UsedPercent)

	return nil
}

func (m *memCollector) update(key string, value float64) {
	if err := m.m.Update(key, "mem", aggregated.AverageMode, value); err != nil {
		log.Error().Err(err).Str("metric", key).Msg("failed to update metric")
	}
}
