package vm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

// Boot config struct
type Boot struct {
	Kernel string `json:"kernel_image_path"`
	Initrd string `json:"initrd_path,omitempty"`
	Args   string `json:"boot_args"`
}

// Disk struct
type Disk struct {
	ID         string `json:"drive_id"`
	Path       string `json:"path_on_host"`
	RootDevice bool   `json:"is_root_device"`
	ReadOnly   bool   `json:"is_read_only"`
}

func (d Disk) String() string {
	on := "off"
	if d.ReadOnly {
		on = "on"
	}

	return fmt.Sprintf(`path=%s,readonly=%s`, d.Path, on)
}

// Disks is a list of vm disks
type Disks []Disk

// Interface nic struct
type Interface struct {
	ID  string `json:"iface_id"`
	Tap string `json:"host_dev_name"`
	Mac string `json:"guest_mac,omitempty"`
}

func (i Interface) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("tap=%s", i.Tap))
	if len(i.Mac) > 0 {
		buf.WriteString(fmt.Sprintf(",mac=%s", i.Mac))
	}

	return buf.String()
}

// Interfaces is a list of node interfaces
type Interfaces []Interface

func (i Interfaces) String() string {

	var buf bytes.Buffer
	for _, inf := range i {
		if buf.Len() > 0 {
			buf.WriteRune(' ')
		}

		buf.WriteString(inf.String())
	}

	return buf.String()
}

// MemMib is memory size in mib
type MemMib int64

func (m MemMib) String() string {
	return fmt.Sprintf("size=%dM", int64(m))
}

// CPU type
type CPU uint8

func (c CPU) String() string {
	return fmt.Sprintf("boot=%d", c)
}

// Config struct
type Config struct {
	CPU       CPU    `json:"vcpu_count"`
	Mem       MemMib `json:"mem_size_mib"`
	HTEnabled bool   `json:"ht_enabled"`
}

// Machine struct
type Machine struct {
	ID         string     `json:"id"`
	Boot       Boot       `json:"boot-source"`
	Disks      Disks      `json:"drives"`
	Interfaces Interfaces `json:"network-interfaces"`
	Config     Config     `json:"machine-config"`
	// NoKeepAlive is not used by firecracker, but instead a marker
	// for the vm  mananger to not restart the machine when it stops
	NoKeepAlive bool `json:"no-keep-alive"`
}

// Save saves a machine into a file
func (m *Machine) Save(n string) error {
	f, err := os.Create(n)
	if err != nil {
		return errors.Wrap(err, "failed to create vm config file")
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(m); err != nil {
		return errors.Wrap(err, "failed to serialize machine object")
	}

	return nil
}

// MachineFromFile loads a vm config from file
func MachineFromFile(n string) (*Machine, error) {
	f, err := os.Open(n)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open machine config file")
	}
	defer f.Close()
	var m Machine
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, errors.Wrap(err, "failed to decode machine config file")
	}

	return &m, nil
}

func (m *Machine) root(base string) string {
	return filepath.Join(base, "firecracker", m.ID, "root")
}

func mount(src, dest string) error {
	if filepath.Clean(src) == filepath.Clean(dest) {
		// nothing to do here
		return nil
	}

	f, err := os.Create(dest)
	if err != nil {
		return errors.Wrapf(err, "failed to touch file: %s", dest)
	}

	f.Close()

	if err := syscall.Mount(src, dest, "", syscall.MS_BIND, ""); err != nil {
		return errors.Wrapf(err, "failed to mount '%s' > '%s'", src, dest)
	}

	return nil
}
