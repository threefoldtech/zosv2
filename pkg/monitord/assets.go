package monitord

import (
	"os"
	"path/filepath"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/net"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
)

// NodeAssets is a struct that hold all the node's hardware information
type NodeAssets struct {
	Host              *host.InfoStat       `json:"host"`
	CPUS              []cpu.InfoStat       `json:"cpus"`
	Drives            map[string]*diskStat `json:"drives"`
	NetworkInterfaces []net.InterfaceStat  `json:"interfaces"`
	Memory            *dmi.DMI             `json:"memory"`
}

type diskStat struct {
	Path   string `json:"path"`
	Fstype string `json:"fstype"`
	Total  uint64 `json:"total"`
}

func getAssets() (NodeAssets, error) {
	nodeAssets := NodeAssets{}

	hostStat, err := host.Info()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.Host = hostStat

	nodeAssets.Drives = make(map[string]*diskStat)
	err = filepath.Walk("/dev", func(path string, info os.FileInfo, err error) error {
		diskUsageStat, err := disk.Usage(path)
		if err != nil {
			return err
		}
		nodeAssets.Drives[path] = &diskStat{
			Path:   diskUsageStat.Path,
			Fstype: diskUsageStat.Fstype,
			Total:  diskUsageStat.Total,
		}

		return nil
	})
	if err != nil {
		return nodeAssets, err
	}

	cpuStat, err := cpu.Info()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.CPUS = cpuStat

	interfStat, err := net.Interfaces()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.NetworkInterfaces = interfStat

	dmi, err := dmi.Decode()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.Memory = dmi

	return nodeAssets, nil
}
