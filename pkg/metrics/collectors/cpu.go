package collectors

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

// CPUCollector type
type cpuCollector struct {
	m metrics.CPU

	keys []Metric
}

// NewCPUCollector created a disk collector
func NewCPUCollector(storage metrics.Storage) Collector {
	return &cpuCollector{
		m: storage,
		keys: []Metric{
			{"node.cpu.used-percent", "cpu usage percent"},
			{"node.cpu.idle", "ideal cpu time per second"},
			{"node.cpu.iowait", "io-wait cpu time per second"},
			{"node.cpu.system", "system cpu time per second"},
			{"node.cpu.irq", "IRQ cpu time per second"},
			{"node.cpu.user", "user cpu time per second"},
		},
	}
}

func (d *cpuCollector) collectCPUs() error {
	cpuUsedPercentStats, err := cpu.Percent(0, true)
	if err != nil {
		return errors.Wrap(err, "failed to get cpu usage percentages")
	}

	for index, cpuPercentStat := range cpuUsedPercentStats {
		d.m.Update("node.cpu.used-percent", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuPercentStat)
	}

	cpuTimes, err := cpu.Times(true)
	if err != nil {
		return errors.Wrap(err, "failed to get cpu time stats")
	}

	for index, cpuTime := range cpuTimes {
		name := fmt.Sprintf("%d", index)

		d.updateDiff("node.cpu.idle", name, cpuTime.Idle)
		d.updateDiff("node.cpu.iowait", name, cpuTime.Iowait)
		d.updateDiff("node.cpu.system", name, cpuTime.System)
		d.updateDiff("node.cpu.irq", name, cpuTime.Irq)
		d.updateDiff("node.cpu.user", name, cpuTime.User)
	}

	return nil
}

func (d *cpuCollector) updateDiff(name, id string, value float64) {
	if err := d.m.Update(name, id, aggregated.DifferentialMode, value); err != nil {
		log.Error().Err(err).Str("metric", name).Str("id", id).Msg("failed to update metric")
	}
}

func (d *cpuCollector) Metrics() []Metric {
	return d.keys
}

// Collect method
func (d *cpuCollector) Collect() error {
	return d.collectCPUs()
}
