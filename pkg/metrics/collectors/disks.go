package collectors

import (
	"github.com/pkg/errors"
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

	d.m.Update("node.pool.size", pool.Label, aggregated.AverageMode, float64(usage.Total))
	d.m.Update("node.pool.used", pool.Label, aggregated.AverageMode, float64(usage.Used))
	d.m.Update("node.pool.free", pool.Label, aggregated.AverageMode, float64(usage.Free))
	d.m.Update("node.pool.used-percent", pool.Label, aggregated.AverageMode, float64(usage.UsedPercent))

	_, err = disk.IOCounters(pool.Devices...)
	if err != nil {
		return errors.Wrapf(err, "failed to get io counters for devices '%+v'", pool.Devices)
	}
	return nil
}

func (d *DiskCollector) collectUnmountedPool(pool *pkg.Pool) error {
	return nil
}

func (d *DiskCollector) collectPools(storage *stubs.StorageModuleStub) error {

	for _, pool := range storage.Pools() {
		if pool.Mounted {
			if err := d.collectMountedPool(&pool); err != nil {
				return err
			}
		} else {
			if err := d.collectUnmountedPool(&pool); err != nil {
				return err
			}
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
