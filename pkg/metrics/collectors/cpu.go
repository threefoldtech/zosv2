package collectors

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

// CPUCollector type
type cpuCollector struct {
	m metrics.CPU

	keys []string
}

// NewCPUCollector created a disk collector
func NewCPUCollector(storage metrics.Storage) Collector {
	return &cpuCollector{
		m: storage,
		keys: []string{
			"node.cpu.used-percent",
			"node.cpu.idle",
			"node.cpu.iowait",
			"node.cpu.system",
			"node.cpu.irq",
			"node.cpu.user",
			"node.cpu.temp",
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
		d.m.Update("node.cpu.idle", fmt.Sprintf("%d", index), aggregated.DifferentialMode, cpuTime.Idle)
		d.m.Update("node.cpu.iowait", fmt.Sprintf("%d", index), aggregated.DifferentialMode, cpuTime.Iowait)
		d.m.Update("node.cpu.system", fmt.Sprintf("%d", index), aggregated.DifferentialMode, cpuTime.System)
		d.m.Update("node.cpu.irq", fmt.Sprintf("%d", index), aggregated.DifferentialMode, cpuTime.Irq)
		d.m.Update("node.cpu.user", fmt.Sprintf("%d", index), aggregated.DifferentialMode, cpuTime.User)
	}

	tempStats, err := host.SensorsTemperatures()
	if err != nil {
		return errors.Wrap(err, "failed to get temperature stats")
	}

	for _, tempStat := range tempStats {
		d.m.Update("node.cpu.temp", tempStat.SensorKey, aggregated.AverageMode, tempStat.Temperature)
	}

	return nil
}

func (d *cpuCollector) Metrics() []string {
	return d.keys
}

// Collect method
func (d *cpuCollector) Collect() error {
	return d.collectCPUs()
}
