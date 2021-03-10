package provision

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"

	"github.com/rs/zerolog/log"
)

// EngineOption interface
type EngineOption interface {
	apply(e *NativeEngine)
}

// WithJanitor sets a janitor for the engine.
// a janitor is executed periodically to clean up
// the deployed resources.
func WithJanitor(j Janitor) EngineOption {
	return &withJanitorOpt{j}
}

// WithUsers sets the user key getter on the
// engine
func WithUsers(g Users) EngineOption {
	return &withUserKeyGetter{g}
}

// WithAdmins sets the admins key getter on the
// engine
func WithAdmins(g Users) EngineOption {
	return &withAdminsKeyGetter{g}
}

// WithStartupOrder forces a specific startup order of types
// any type that is not listed in this list, will get started
// in an nondeterministic order
func WithStartupOrder(t ...gridtypes.WorkloadType) EngineOption {
	return &withStartupOrder{t}
}

type provisionJob struct {
	wl gridtypes.Deployment
	ch chan error
}

type deprovisionJob struct {
	twin   uint32
	id     uint32
	ch     chan error
	reason string
}

// NativeEngine is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type NativeEngine struct {
	storage     Storage
	provisioner Provisioner

	provision   chan provisionJob
	deprovision chan deprovisionJob

	//options
	// janitor Janitor
	users  Users
	admins Users
	order  []gridtypes.WorkloadType
}

var _ Engine = (*NativeEngine)(nil)
var _ pkg.Provision = (*NativeEngine)(nil)

type withJanitorOpt struct {
	j Janitor
}

func (o *withJanitorOpt) apply(e *NativeEngine) {
	panic("not imple=nted")
	// e.janitor = o.j
}

type withUserKeyGetter struct {
	g Users
}

func (o *withUserKeyGetter) apply(e *NativeEngine) {
	e.users = o.g
}

type withAdminsKeyGetter struct {
	g Users
}

func (o *withAdminsKeyGetter) apply(e *NativeEngine) {
	e.admins = o.g
}

type withStartupOrder struct {
	o []gridtypes.WorkloadType
}

func (w *withStartupOrder) apply(e *NativeEngine) {
	all := make(map[gridtypes.WorkloadType]struct{})
	for _, typ := range e.order {
		all[typ] = struct{}{}
	}
	ordered := make([]gridtypes.WorkloadType, 0, len(all))
	for _, typ := range w.o {
		if _, ok := all[typ]; !ok {
			panic(fmt.Sprintf("type '%s' is not registered", typ))
		}
		delete(all, typ)
		ordered = append(ordered, typ)
	}
	// now move everything else
	for typ := range all {
		ordered = append(ordered, typ)
	}

	e.order = ordered
}

type nullKeyGetter struct{}

func (n *nullKeyGetter) GetKey(id gridtypes.ID) (ed25519.PublicKey, error) {
	return nil, fmt.Errorf("null user key getter")
}

type engineKey struct{}
type deploymentKey struct{}

// GetEngine gets engine from context
func GetEngine(ctx context.Context) Engine {
	return ctx.Value(engineKey{}).(Engine)
}

// GetDeployment gets a copy of the current deployment
func GetDeployment(ctx context.Context) gridtypes.Deployment {
	return ctx.Value(deploymentKey{}).(gridtypes.Deployment)
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(storage Storage, provisioner Provisioner, opts ...EngineOption) *NativeEngine {

	e := &NativeEngine{
		storage:     storage,
		provisioner: provisioner,
		provision:   make(chan provisionJob),
		deprovision: make(chan deprovisionJob),
		users:       &nullKeyGetter{},
		admins:      &nullKeyGetter{},
		order:       gridtypes.Types(),
	}

	for _, opt := range opts {
		opt.apply(e)
	}

	return e
}

// Storage returns
func (e *NativeEngine) Storage() Storage {
	return e.storage
}

// Users returns users db
func (e *NativeEngine) Users() Users {
	return e.users
}

// Admins returns admins db
func (e *NativeEngine) Admins() Users {
	return e.admins
}

// Provision workload
func (e *NativeEngine) Provision(ctx context.Context, deployment gridtypes.Deployment) error {
	j := provisionJob{
		wl: deployment,
		ch: make(chan error),
	}

	defer close(j.ch)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.provision <- j:
		return <-j.ch
	}
}

// Deprovision workload
func (e *NativeEngine) Deprovision(ctx context.Context, twin, id uint32, reason string) error {
	j := deprovisionJob{
		twin:   twin,
		id:     id,
		ch:     make(chan error),
		reason: reason,
	}

	defer close(j.ch)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.deprovision <- j:
		return <-j.ch
	}
}

// Run starts reader reservation from the Source and handle them
func (e *NativeEngine) Run(ctx context.Context) error {
	defer close(e.provision)
	defer close(e.deprovision)

	ctx = context.WithValue(ctx, engineKey{}, e)

	// restart everything first
	// TODO: potential network disconnections if network already exists.
	// may be network manager need to do nothing if same exact network config
	// is applied
	if err := e.boot(ctx); err != nil {
		log.Error().Err(err).Msg("error while setting up")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case job := <-e.deprovision:
			dl, err := e.storage.Get(job.twin, job.id)
			if err != nil {
				job.ch <- err
				log.Error().Err(err).Uint32("twin", dl.TwinID).Uint32("id", job.id).Msg("failed to get workload from storage")
				continue
			}
			//wl.ToDelete = true
			err = e.storage.Set(dl)
			job.ch <- err
			if err != nil {
				log.Error().Err(err).Uint32("id", job.id).Msg("failed to mark workload at to be delete")
				continue
			}

			e.uninstallDeployment(ctx, dl, job.reason)
		case job := <-e.provision:
			deployment := job.wl

			//TODO: check for migration (from older version to new)
			err := e.storage.Add(deployment)
			// NOTE: hack to force reinstall of same reservation
			// TODO: remove hack
			if errors.Is(err, ErrDeploymentExists) {
				err = e.storage.Set(deployment)
			}
			// release the job. the caller will now know that the workload
			// has been committed to storage (or not)
			job.ch <- err
			if err != nil {
				log.Error().Err(err).Uint32("twin", job.wl.TwinID).Uint32("id", job.wl.DeploymentID).Msg("failed to commit deployment to storage")
				continue
			}

			e.installDeployment(ctx, deployment)
		}
	}
}

// boot will make sure to re-deploy all stored reservation
// on boot.
func (e *NativeEngine) boot(ctx context.Context) error {
	storage := e.Storage()
	twins, err := storage.Twins()
	if err != nil {
		return errors.Wrap(err, "failed to list twins")
	}
	for _, twin := range twins {
		ids, err := storage.ByTwin(twin)
		if err != nil {
			log.Error().Err(err).Uint32("twin", twin).Msg("failed to list deployments for twin")
			continue
		}

		for _, id := range ids {
			dl, err := storage.Get(twin, id)
			if err != nil {
				log.Error().Err(err).Uint32("twin", twin).Uint32("id", id).Msg("failed to load deployment")
				continue
			}

			e.installDeployment(ctx, dl)
		}
	}

	return nil
}

func (e *NativeEngine) uninstallWorkload(ctx context.Context, wl *gridtypes.WorkloadWithID, reason string) error {
	err := e.provisioner.Decommission(ctx, wl)
	result := &gridtypes.Result{
		Error: reason,
		State: gridtypes.StateDeleted,
	}

	if err != nil {
		log.Error().Err(err).Stringer("global-id", wl.ID).Msg("failed to uninstall workload")
		result.State = gridtypes.StateError
		result.Error = errors.Wrapf(err, "error while decommission reservation because of: '%s'", result.Error).Error()
	}

	result.Created = gridtypes.Timestamp(time.Now().Unix())
	wl.Result = *result
	return err
}

func (e *NativeEngine) uninstallDeployment(ctx context.Context, dl gridtypes.Deployment, reason string) {
	log := log.With().Uint32("twin", dl.TwinID).Uint32("id", dl.DeploymentID).Logger()
	ctx = context.WithValue(ctx, deploymentKey{}, dl)

	for i := len(e.order) - 1; i >= 0; i-- {
		typ := e.order[i]

		workloads := dl.ByType(typ)
		for _, wl := range workloads {

			log.Debug().Str("workload", wl.Name).Msg("de-provisioning")
			if wl.Result.State == gridtypes.StateDeleted ||
				wl.Result.State == gridtypes.StateError {
				//nothing to do!
				continue
			}

			_ = e.uninstallWorkload(ctx, wl, reason)

			if err := e.storage.Set(dl); err != nil {
				log.Error().Err(err).Msg("failed to set workload result")
			}
		}
	}
}

func (e *NativeEngine) installDeployment(ctx context.Context, deployment gridtypes.Deployment) {
	log := log.With().Uint32("twin", deployment.TwinID).Uint32("id", deployment.DeploymentID).Logger()
	ctx = context.WithValue(ctx, deploymentKey{}, deployment)

	for _, typ := range e.order {
		workloads := deployment.ByType(typ)

		for _, wl := range workloads {
			log := log.With().Str("type", wl.Type.String()).Str("name", wl.Name).Logger()
			log.Debug().Msg("provisioning")

			result, err := e.provisioner.Provision(ctx, wl)
			if err != nil {
				log.Error().Err(err).Msg("failed to deploy workload")
				result = &gridtypes.Result{
					Error: err.Error(),
					State: gridtypes.StateError,
				}
			}

			if result.State == gridtypes.StateError {
				log.Error().Str("error", result.Error).Msg("failed to deploy workload")
			}

			result.Created = gridtypes.Timestamp(time.Now().Unix())
			wl.Result = *result

			if err := e.storage.Set(deployment); err != nil {
				log.Error().Err(err).Msg("failed to set workload result")
			}
		}
	}
}

// Counters implements the zbus interface
func (e *NativeEngine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	//TODO: implement counters
	// this is probably need to be moved to
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			ch <- pkg.ProvisionCounters{}
		}
	}()

	return ch
}

// DecommissionCached implements the zbus interface
func (e *NativeEngine) DecommissionCached(id string, reason string) error {
	globalID := gridtypes.WorkloadID(id)
	twin, dlID, name, err := globalID.Parts()
	if err != nil {
		return err
	}
	dl, err := e.storage.Get(twin, dlID)
	if err != nil {
		return err
	}

	wl, err := dl.Get(name)
	if err != nil {
		return err
	}

	if wl.Result.State == gridtypes.StateDeleted ||
		wl.Result.State == gridtypes.StateError {
		//nothing to do!
		return nil
	}

	//to bad we have to repeat this here
	ctx := context.WithValue(context.Background(), engineKey{}, e)
	ctx = context.WithValue(ctx, deploymentKey{}, dl)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	err = e.uninstallWorkload(ctx, wl, reason)

	if err := e.storage.Set(dl); err != nil {
		log.Error().Err(err).Msg("failed to set workload result")
	}

	return err
}
