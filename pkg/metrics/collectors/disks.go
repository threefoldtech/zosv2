package collectors

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/disk"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
	"github.com/threefoldtech/zos/pkg/stubs"
)

type DiskCollector struct {
	cl zbus.Client
	m  metrics.Storage
}

func (d *DiskCollector) collectMountedPool(pool *pkg.Pool) error {
	usage, err := disk.Usage(pool.Path)
	if err != nil {
		return errors.Wrapf(err, "failed to get usage of mounted pool '%s'", pool.Path)
	}

	d.updateAvg("node.pool.mounted", pool.Label, 1)

	d.updateAvg("node.pool.size", pool.Label, float64(usage.Total))
	d.updateAvg("node.pool.used", pool.Label, float64(usage.Used))
	d.updateAvg("node.pool.free", pool.Label, float64(usage.Free))
	d.updateAvg("node.pool.used-percent", pool.Label, float64(usage.UsedPercent))

	counters, err := disk.IOCounters(pool.Devices...)
	if err != nil {
		return errors.Wrapf(err, "failed to get io counters for devices '%+v'", pool.Devices)
	}

	for disk, counter := range counters {
		d.updateDiff("node.disk.read-bytes", disk, float64(counter.ReadBytes))
		d.updateDiff("node.disk.read-count", disk, float64(counter.ReadCount))
		d.updateDiff("node.disk.read-time", disk, float64(counter.ReadTime))
		d.updateDiff("node.disk.write-bytes", disk, float64(counter.ReadBytes))
		d.updateDiff("node.disk.write-count", disk, float64(counter.ReadCount))
		d.updateDiff("node.disk.write-time", disk, float64(counter.ReadTime))
	}

	return nil
}

func (d *DiskCollector) updateAvg(name, id string, value float64) {
	if err := d.m.Update(name, id, aggregated.AverageMode, value); err != nil {
		log.Error().Err(err).Msgf("failed to update metric '%s:%s'", name, id)
	}
}

func (d *DiskCollector) updateDiff(name, id string, value float64) {
	if err := d.m.Update(name, id, aggregated.AverageMode, value); err != nil {
		log.Error().Err(err).Msgf("failed to update metric '%s:%s'", name, id)
	}
}
func (d *DiskCollector) collectUnmountedPool(pool *pkg.Pool) error {
	d.updateAvg("node.pool.mounted", pool.Label, 1)

	for _, device := range pool.Devices {
		disk := filepath.Base(device)
		d.updateDiff("node.disk.read-bytes", disk, 0)
		d.updateDiff("node.disk.read-count", disk, 0)
		d.updateDiff("node.disk.read-time", disk, 0)
		d.updateDiff("node.disk.write-bytes", disk, 0)
		d.updateDiff("node.disk.write-count", disk, 0)
		d.updateDiff("node.disk.write-time", disk, 0)
	}
}

func (d *DiskCollector) collectPools(storage *stubs.StorageModuleStub) error {

	for _, pool := range storage.Pools() {
		collector := d.collectMountedPool

		if !pool.Mounted {
			collector = d.collectUnmountedPool
		}

		if err := collector(&pool); err != nil {

		}
	}

	return nil
}

// Collect method
func (d *DiskCollector) Collect() error {
	// - we list the pools and device from stroaged
	// - to get usage information we need to access pool.Path (/mnt/<id>)
	//   - (we know its btrfs)
	// - for mounted pools
	//   - check each device IO counters
	// - for broken pools
	//   - device.broken (1 or 0)
	storage := stubs.NewStorageModuleStub(d.cl)
	d.collectPools(storage)
	return nil
}
