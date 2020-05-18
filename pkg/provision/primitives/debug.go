package primitives

import (
	"context"
	"encoding/json"
	"os"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"

	"github.com/pkg/errors"
)

type DebugInfo struct {
	Sysdiag bool `json:"sysdiag"`
}

type DebugResult struct {
	ResultFile string `json:"result"`
}

func (p *Provisioner) debugSysdiag() (DebugResult, error) {
	log.Debug().Msg("preparing sysdiag request")

	fl := "https://hub.grid.tf/maxux42.3bot/sysdiag.flist"
	st := "zdb://hub.grid.tf:9900"

	flistClient := stubs.NewFlisterStub(p.zbus)

	log.Debug().Str("flist", fl).Msg("sysdiag flist")
	var mnt string

	mnt, err := flistClient.Mount(fl, st, pkg.DefaultMountOptions)
	if err != nil {
		return DebugResult{}, err
	}

	defer func() {
		log.Debug().Msg("cleaning sysdiag flist")

		if err := flistClient.Umount(mnt); err != nil {
			log.Error().Err(err).Str("mnt", mnt).Msg("failed to unmount sysdiag root")
		}
	}()

	log.Info().Str("rootfs", mnt).Msg("initializing sysdiag environment")

	env := os.Environ()
	args := []string{"sh", mnt + "/sysdiag.sh"}

	execErr := syscall.Exec("/bin/sh", args, env)
	if execErr != nil {
		return DebugResult{}, errors.Wrap(err, "error executing sysdiag")
	}

	rslt := DebugResult{
		ResultFile: "unknown for now",
	}

	return rslt, nil
}

func (p *Provisioner) debugProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.debugProvisionImpl(ctx, reservation)
}

// ContainerProvision is entry point to container reservation
func (p *Provisioner) debugProvisionImpl(ctx context.Context, reservation *provision.Reservation) (DebugResult, error) {
	var config DebugInfo
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return DebugResult{}, err
	}

	log.Info().Bool("sysdiag", config.Sysdiag).Msg("debug request")

	if config.Sysdiag {
		return p.debugSysdiag()
	}

	// not reached for now
	result := DebugResult{
		ResultFile: "coucou",
	}

	return result, nil
}

func (p *Provisioner) debugDecommission(ctx context.Context, reservation *provision.Reservation) error {
	return nil
}

/* OLD CODE */

/*
// Debug provision schema
type Debug struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Channel string `json:"channel"`
}

func (p *Provisioner) debugProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	var cfg Debug
	if err := json.Unmarshal(reservation.Data, &cfg); err != nil {
		return nil, err
	}

	_, err := p.startZLF(ctx, reservation.ID, cfg)
	// nothing to return to BCDB
	return nil, err
}

func (p *Provisioner) debugDecommission(ctx context.Context, reservation *provision.Reservation) error {
	return p.stopZLF(ctx, reservation.ID)
}

func (p *Provisioner) startZLF(ctx context.Context, ID string, cfg Debug) (string, error) {
	identity := stubs.NewIdentityManagerStub(p.zbus)

	path, err := exec.LookPath("zlf")
	if err != nil {
		return "", errors.Wrap(err, "failed to start zlf")
	}

	z, err := zinit.New("")
	if err != nil {
		return "", errors.Wrap(err, "fail to connect to zinit")
	}
	defer z.Close()

	channel := fmt.Sprintf("%s-logs", identity.NodeID().Identity())
	if cfg.Channel != "" {
		channel = cfg.Channel
	}

	s := zinit.InitService{
		Exec:    fmt.Sprintf("%s --host %s --port %d --channel %s", path, cfg.Host, cfg.Port, channel),
		Oneshot: false,
		After:   []string{"networkd"},
		Log:     zinit.StdoutLogType,
	}

	name := fmt.Sprintf("zlf-debug-%s", ID)
	if err := zinit.AddService(name, s); err != nil {
		return "", errors.Wrap(err, "fail to add init service to zinit")
	}

	if err := z.Monitor(name); err != nil {
		return "", errors.Wrap(err, "failed to start monitoring zlf service")
	}

	return name, nil
}

func (p *Provisioner) stopZLF(ctx context.Context, ID string) error {
	z, err := zinit.New("")
	if err != nil {
		return errors.Wrap(err, "fail to connect to zinit")
	}
	defer z.Close()

	name := fmt.Sprintf("zlf-debug-%s", ID)
	services, err := z.List()
	if err != nil {
		return errors.Wrap(err, "failed to list zinit services")
	}
	found := false
	for s := range services {
		if strings.Contains(s, name) {
			found = true
			break
		}
	}
	if !found {
		log.Info().Str("service", name).Msg("zinit service not found, nothing else to do")
		return nil
	}

	if err := z.Stop(name); err != nil {
		return errors.Wrapf(err, "failed to stop %s zlf service", name)
	}

	if err := z.Forget(name); err != nil {
		return errors.Wrapf(err, "failed to forget %s zlf service", name)
	}

	if err := zinit.RemoveService(name); err != nil {
		return errors.Wrapf(err, "failed to delete %s zlf service", name)
	}

	return nil
}
*/
