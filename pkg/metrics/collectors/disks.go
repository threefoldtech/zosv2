package collectors

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
	"github.com/threefoldtech/zos/pkg/stubs"
)

type diskCollector struct {
	cl zbus.Client
	m  metrics.Storage

	keys []Metric
}

// NewDiskCollector created a disk collector
func NewDiskCollector(cl zbus.Client, storage metrics.Storage) Collector {
	return &diskCollector{
		cl: cl,
		m:  storage,
		keys: []Metric{
			{"health.pool.mounted", "pool is mounted (1) or not mounted (0)"},
			{"health.pool.broken", "pool is broken (1) or not broken (0)"},
			{"utilization.pool.size", "pool size in bytes"},
			{"utilization.pool.used", "pool used space in bytes"},
			{"utilization.pool.free", "pool free space in bytes"},
			{"utilization.pool.used-percent", "pool usage percent"},
			{"utilization.disk.read-bytes", "average disk read bytes per second"},
			{"utilization.disk.read-count", "average number of read operations per second"},
			{"utilization.disk.read-time", "average read operation time per second"},
			{"utilization.disk.write-bytes", "average disk read bytes per second"},
			{"utilization.disk.write-count", "average number of write operations per second"},
			{"utilization.disk.write-time", "average number of write operations per second"},
		},
	}
}

func (d *diskCollector) collectMountedPool(pool *pkg.Pool) error {
	usage, err := disk.Usage(pool.Path)
	if err != nil {
		return errors.Wrapf(err, "failed to get usage of mounted pool '%s'", pool.Path)
	}

	d.updateAvg("health.pool.mounted", pool.Label, 1)
	d.updateAvg("health.pool.broken", pool.Label, 0)

	d.updateAvg("utilization.pool.size", pool.Label, float64(usage.Total))
	d.updateAvg("utilization.pool.used", pool.Label, float64(usage.Used))
	d.updateAvg("utilization.pool.free", pool.Label, float64(usage.Free))
	d.updateAvg("utilization.pool.used-percent", pool.Label, float64(usage.UsedPercent))

	counters, err := disk.IOCounters(pool.Devices...)
	if err != nil {
		return errors.Wrapf(err, "failed to get io counters for devices '%+v'", pool.Devices)
	}

	for disk, counter := range counters {
		d.updateAvg("health.disk.broken", disk, 0)

		d.updateDiff("utilization.disk.read-bytes", disk, float64(counter.ReadBytes))
		d.updateDiff("utilization.disk.read-count", disk, float64(counter.ReadCount))
		d.updateDiff("utilization.disk.read-time", disk, float64(counter.ReadTime))
		d.updateDiff("utilization.disk.write-bytes", disk, float64(counter.ReadBytes))
		d.updateDiff("utilization.disk.write-count", disk, float64(counter.ReadCount))
		d.updateDiff("utilization.disk.write-time", disk, float64(counter.ReadTime))
	}

	return nil
}

func (d *diskCollector) updateAvg(name, id string, value float64) {
	if err := d.m.Update(name, id, aggregated.AverageMode, value); err != nil {
		log.Error().Err(err).Str("metric", name).Str("id", id).Msg("failed to update metric")
	}
}

func (d *diskCollector) updateDiff(name, id string, value float64) {
	if err := d.m.Update(name, id, aggregated.DifferentialMode, value); err != nil {
		log.Error().Err(err).Str("metric", name).Str("id", id).Msg("failed to update metric")
	}
}
func (d *diskCollector) collectUnmountedPool(pool *pkg.Pool) error {
	d.updateAvg("health.pool.mounted", pool.Label, 0)
	d.updateAvg("health.pool.broken", pool.Label, 0)

	for _, device := range pool.Devices {
		disk := filepath.Base(device)
		d.updateAvg("utilization.disk.broken", disk, 0)
		d.updateDiff("utilization.disk.read-bytes", disk, 0)
		d.updateDiff("utilization.disk.read-count", disk, 0)
		d.updateDiff("utilization.disk.read-time", disk, 0)
		d.updateDiff("utilization.disk.write-bytes", disk, 0)
		d.updateDiff("utilization.disk.write-count", disk, 0)
		d.updateDiff("utilization.disk.write-time", disk, 0)
	}

	return nil
}

func (d *diskCollector) collectPools(storage *stubs.StorageModuleStub) {
	for _, pool := range storage.Pools() {
		collector := d.collectMountedPool

		if !pool.Mounted {
			collector = d.collectUnmountedPool
		}

		if err := collector(&pool); err != nil {
			log.Error().Err(err).Str("pool", pool.Label).Msg("failed to collect metrics for pool")
		}
	}
}

func (d *diskCollector) collectBrokenPools(storage *stubs.StorageModuleStub) {
	for _, pool := range storage.BrokenPools() {
		d.updateAvg("utilization.pool.mounted", pool.Label, 0)
		d.updateAvg("utilization.pool.broken", pool.Label, 1)
	}

	for _, device := range storage.BrokenDevices() {
		disk := filepath.Base(device.Path)
		d.updateAvg("utilization.disk.broken", disk, 1)
	}
}

func (d *diskCollector) Metrics() []Metric {
	return d.keys
}

// Collect method
func (d *diskCollector) Collect() error {
	// - we list the pools and device from stroaged
	// - to get usage information we need to access pool.Path (/mnt/<id>)
	//   - (we know its btrfs)
	// - for mounted pools
	//   - check each device IO counters
	// - for broken pools
	//   - device.broken (1 or 0)
	storage := stubs.NewStorageModuleStub(d.cl)
	d.collectPools(storage)
	d.collectBrokenPools(storage)

	return nil
}
