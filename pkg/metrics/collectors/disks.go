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

type diskCollector struct {
	cl zbus.Client
	m  metrics.Storage

	keys []string
}

// NewDiskCollector created a disk collector
func NewDiskCollector(cl zbus.Client, storage metrics.Storage) Collector {
	return &diskCollector{
		cl: cl,
		m:  storage,
		keys: []string{
			"node.pool.mounted",
			"node.pool.broken",
			"node.pool.size",
			"node.pool.used",
			"node.pool.free",
			"node.pool.used-percent",
			"node.disk.read-bytes",
			"node.disk.read-count",
			"node.disk.read-time",
			"node.disk.write-bytes",
			"node.disk.write-count",
			"node.disk.write-time",
		},
	}
}

func (d *diskCollector) collectMountedPool(pool *pkg.Pool) error {
	usage, err := disk.Usage(pool.Path)
	if err != nil {
		return errors.Wrapf(err, "failed to get usage of mounted pool '%s'", pool.Path)
	}

	d.updateAvg("node.pool.mounted", pool.Label, 1)
	d.updateAvg("node.pool.broken", pool.Label, 0)

	d.updateAvg("node.pool.size", pool.Label, float64(usage.Total))
	d.updateAvg("node.pool.used", pool.Label, float64(usage.Used))
	d.updateAvg("node.pool.free", pool.Label, float64(usage.Free))
	d.updateAvg("node.pool.used-percent", pool.Label, float64(usage.UsedPercent))

	counters, err := disk.IOCounters(pool.Devices...)
	if err != nil {
		return errors.Wrapf(err, "failed to get io counters for devices '%+v'", pool.Devices)
	}

	for disk, counter := range counters {
		d.updateAvg("node.disk.broken", disk, 0)
		d.updateDiff("node.disk.read-bytes", disk, float64(counter.ReadBytes))
		d.updateDiff("node.disk.read-count", disk, float64(counter.ReadCount))
		d.updateDiff("node.disk.read-time", disk, float64(counter.ReadTime))
		d.updateDiff("node.disk.write-bytes", disk, float64(counter.ReadBytes))
		d.updateDiff("node.disk.write-count", disk, float64(counter.ReadCount))
		d.updateDiff("node.disk.write-time", disk, float64(counter.ReadTime))
	}

	return nil
}

func (d *diskCollector) updateAvg(name, id string, value float64) {
	if err := d.m.Update(name, id, aggregated.AverageMode, value); err != nil {
		log.Error().Err(err).Msgf("failed to update metric '%s:%s'", name, id)
	}
}

func (d *diskCollector) updateDiff(name, id string, value float64) {
	if err := d.m.Update(name, id, aggregated.AverageMode, value); err != nil {
		log.Error().Err(err).Msgf("failed to update metric '%s:%s'", name, id)
	}
}
func (d *diskCollector) collectUnmountedPool(pool *pkg.Pool) error {
	d.updateAvg("node.pool.mounted", pool.Label, 0)
	d.updateAvg("node.pool.broken", pool.Label, 0)

	for _, device := range pool.Devices {
		disk := filepath.Base(device)
		d.updateAvg("node.disk.broken", disk, 0)
		d.updateDiff("node.disk.read-bytes", disk, 0)
		d.updateDiff("node.disk.read-count", disk, 0)
		d.updateDiff("node.disk.read-time", disk, 0)
		d.updateDiff("node.disk.write-bytes", disk, 0)
		d.updateDiff("node.disk.write-count", disk, 0)
		d.updateDiff("node.disk.write-time", disk, 0)
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
		d.updateAvg("node.pool.mounted", pool.Label, 0)
		d.updateAvg("node.pool.broken", pool.Label, 1)
	}

	for _, device := range storage.BrokenDevices() {
		disk := filepath.Base(device.Path)
		d.updateAvg("node.disk.broken", disk, 1)
	}
}

func (d *diskCollector) Metrics() []string {
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
