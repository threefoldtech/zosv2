package collectors

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/cpu"
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
			{"utilization.cpu.used-percent", "cpu usage percent"},
			{"utilization.cpu.idle", "percent of ideal cpu time"},
			{"utilization.cpu.iowait", "percent of io-wait cpu time"},
			{"utilization.cpu.system", "percent of system cpu time"},
			{"utilization.cpu.irq", "percent of IRQ cpu time"},
			{"utilization.cpu.user", "percent of user cpu time"},
		},
	}
}

func (d *cpuCollector) collectCPUs() error {
	cpuUsedPercentStats, err := cpu.Percent(0, true)
	if err != nil {
		return errors.Wrap(err, "failed to get cpu usage percentages")
	}

	for index, cpuPercentStat := range cpuUsedPercentStats {
		d.m.Update("utilization.cpu.used-percent", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuPercentStat)
	}

	cpuTimes, err := cpu.Times(true)
	if err != nil {
		return errors.Wrap(err, "failed to get cpu time stats")
	}

	for index, cpuTime := range cpuTimes {
		name := fmt.Sprintf("%d", index)
		d.updateDiff("utilization.cpu.idle", name, cpuTime.Idle*100)
		d.updateDiff("utilization.cpu.iowait", name, cpuTime.Iowait*100)
		d.updateDiff("utilization.cpu.system", name, cpuTime.System*100)
		d.updateDiff("utilization.cpu.irq", name, cpuTime.Irq*100)
		d.updateDiff("utilization.cpu.user", name, cpuTime.User*100)
	}

	return nil
}

func (d *cpuCollector) updateDiff(name, id string, value float64) {
	log.Debug().Str("metric", name).Str("id", id).Float64("value", value).Msg("reported")
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
