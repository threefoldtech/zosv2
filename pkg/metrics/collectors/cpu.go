package collectors

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

// CPUCollector type
type CPUCollector struct {
	cl zbus.Client
	m  metrics.CPU
}

func (d *CPUCollector) collectCPUs() error {
	cpuUsedPercentStats, err := cpu.Percent(0*time.Second, true)
	if err != nil {
		return errors.Wrap(err, "failed to get cpu usage percentages")
	}

	for index, cpuPercentStat := range cpuUsedPercentStats {
		d.m.Update("node.cpu.used", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuPercentStat)
	}

	cpuTimes, err := cpu.Times(true)
	if err != nil {
		return errors.Wrap(err, "failed to get cpu time stats")
	}

	for index, cpuTime := range cpuTimes {
		d.m.Update("node.cpu.idle", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuTime.Idle)
		d.m.Update("node.cpu.iowait", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuTime.Iowait)
		d.m.Update("node.cpu.system", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuTime.System)
		d.m.Update("node.cpu.irq", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuTime.Irq)
		d.m.Update("node.cpu.user", fmt.Sprintf("%d", index), aggregated.AverageMode, cpuTime.User)
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

// Collect method
func (d *CPUCollector) Collect() error {
	return d.collectCPUs()
}
