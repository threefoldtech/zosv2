package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
)

// QemuResult result returned by qemu reservation
type QemuResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// Qemu reservation data
type Qemu struct {
	// NetworkID of the network namepsace in which to run the VM. The network
	// must be provisioned previously.
	NetworkID pkg.NetID `json:"network_id"`
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IP `json:"ip"`
	// Image of the VM.
	Image string `json:"image"`
	// URL of the storage backend for the flist
	ImageFlistStorage string `json:"image_flist_storage"`
	// QemuCapacity is the amount of resource to allocate to the virtual machine
	Capacity QemuCapacity `json:"capacity"`
}

// QemuCapacity is the amount of resource to allocate to the virtual machine
type QemuCapacity struct {
	// Number of CPU
	CPU uint `json:"cpu"`
	// Memory in MiB
	Memory uint64 `json:"memory"`
	// HDD in GB
	HDDSize uint64 `json:"hdd"`
}

const qemuFlistURL = "https://hub.grid.tf/maximevanhees.3bot/qemu-flist-tarball.flist"

func (p *Provisioner) qemuProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.qemuProvisionImpl(ctx, reservation)
}

func (p *Provisioner) qemuProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result QemuResult, err error) {
	qemuClient := stubs.NewQemuModuleStub(p.zbus)
	flistClient := stubs.NewFlisterStub(p.zbus)
	vmClient := stubs.NewVMModuleStub(p.zbus)
	vm := stubs.NewVMModuleStub(p.zbus)

	var needsInstall = true

	// checking reservation scheme and putting data in config
	var config Qemu
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	// mounting image
	var mnt string
	mnt, err = flistClient.Mount(config.Image, config.ImageFlistStorage, pkg.DefaultMountOptions)
	if err != nil {
		return QemuResult{}, err
	}
	// unmount flist if error occurs
	defer func() {
		if err != nil {
			if err := flistClient.Umount(mnt); err != nil {
				log.Error().Err(err).Str("mnt", mnt).Msg("failed to unmount")
			}
		}
	}()

	// set IP address
	result.IP = config.IP.string()
	// set ID
	result.ID = reservation.ID


	cpu, memory, disk, err := vmSize(config.Capacity)
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	var imagePath string
	imagePath, err = flist.NamedMount(reservation.ID, qemuFlistURL, "", pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrap(err, "could not mount qemu flist")
	}
	// In case of future errrors in the provisioning make sure we clean up
	defer func() {
		if err != nil {
			_ = flist.Umount(imagePath)
		}
	}()

	var diskPath string
	diskName := fmt.Sprintf("%s-%s", reservation.ID, "vda")
	if storage.Exists(diskName) {
		needsInstall = false
		info, err := storage.Inspect(diskName)
		if err != nil {
			return result, errors.Wrap(err, "could not get path to existing disk")
		}
		diskName = info.Path
	} else {
		diskPath, err = storage.Allocate(diskName, int64(disk))
		if err != nil {
			return result, errors.Wrap(err, "failed to reserve filesystem for vm")
		}
	}
	// clean up the disk anyway, even if it has already been installed.
	defer func() {
		if err != nil {
			_ = storage.Deallocate(diskName)
		}
	}()

	// setup tap device
	var iface string
	netID := networkID(reservation.User, string(config.NetworkID))
	iface, err = network.SetupTap(netID)
	if err != nil {
		return result, errors.Wrap(err, "could not set up tap device")
	}

	defer func() {
		if err != nil {
			_ = vm.Delete(reservation.ID)
			_ = network.RemoveTap(netID)
		}
	}()

	var netInfo pkg.VMNetworkInfo
	netInfo, err = p.buildNetworkInfo(ctx, reservation.User, iface, config)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}

	if needsInstall {
		if err = p.qemuInstall(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config); err != nil {
			return result, errors.Wrap(err, "failed to install qemu")
		}
	}

	err = p.qemuRun(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config)

	return result, err
}

func (p *Provisioner) qemuInstall(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Qemu) error {
	// prepare disks here
	// ....

	deadline, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()
	for {
		if !vm.Exists(name) {
			// install is done
			break
		}
		select {
		case <-time.After(time.Second * 3):
			// retry after 3 secs
		case <-deadline.Done():
			return errors.New("failed to install vm in 5 minutes")
		}
	}

	// Delete the VM, the disk will be installed now
	return vm.Delete(name)
}

func (p *Provisioner) qemuRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Qemu) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	// create virtual machine and run it
	qemuVM := {}

	return vm.Run(qemuVM)
}



func (p *Provisioner) qemuDecommision(ctx context.Context, reservation *provision.Reservation) error {
	return nil
}


// returns the vCpu's, memory, disksize for a vm size
// memory and disk size is expressed in MiB
// hardcoded for now to test
func vmSize(size uint8) (uint8, uint64, uint64, error) {
	return 1, 2 * 1024, 50 * 1024, nil)
}