package provision

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jbenet/go-base58"
	gsrpc "github.com/leesmet/go-substrate-rpc-client"
	"github.com/leesmet/go-substrate-rpc-client/scale"
	"github.com/leesmet/go-substrate-rpc-client/signature"
	"github.com/leesmet/go-substrate-rpc-client/types"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"

	"gopkg.in/robfig/cron.v2"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Engine is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type Engine struct {
	nodeID         string
	source         ReservationSource
	cache          ReservationCache
	feedback       Feedbacker
	provisioners   map[ReservationType]ProvisionerFunc
	decomissioners map[ReservationType]DecomissionerFunc
	signer         Signer
	statser        Statser
	zbusCl         zbus.Client
}

// EngineOps are the configuration of the engine
type EngineOps struct {
	// NodeID is the identity of the system running the engine
	NodeID string
	// Source is responsible to retrieve reservation for a remote source
	Source ReservationSource
	// Feedback is used to send provision result to the source
	// after the reservation is provisionned
	Feedback Feedbacker
	// Cache is a used to keep track of which reservation are provisionned on the system
	// and know when they expired so they can be decommissioned
	Cache ReservationCache
	// Provisioners is a function map so the engine knows how to provision the different
	// workloads supported by the system running the engine
	Provisioners map[ReservationType]ProvisionerFunc
	// Decomissioners contains the opposite function from Provisioners
	// they are used to decomission workloads from the system
	Decomissioners map[ReservationType]DecomissionerFunc
	// Signer is used to authenticate the result send to the source
	Signer Signer
	// Statser is responsible to keep track of how much workloads and resource units
	// are reserved on the system running the engine
	// After each provision/decomission the engine sends statistics update to the staster
	Statser Statser
	// ZbusCl is a client to Zbus
	ZbusCl zbus.Client
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(opts EngineOps) *Engine {
	return &Engine{
		nodeID:         opts.NodeID,
		source:         opts.Source,
		cache:          opts.Cache,
		feedback:       opts.Feedback,
		provisioners:   opts.Provisioners,
		decomissioners: opts.Decomissioners,
		signer:         opts.Signer,
		statser:        opts.Statser,
		zbusCl:         opts.ZbusCl,
	}
}

// Run starts reader reservation from the Source and handle them
func (e *Engine) Run(ctx context.Context) error {
	identity := stubs.NewIdentityManagerStub(e.zbusCl)
	poller := NewSubstratePoller(identity)
	err := poller.createAddressFromID()
	if err != nil {
		return err
	}
	cReservation := poller.GetReservations(ctx)

	isAllWorkloadsProcessed := false
	// run a cron task that will fire the cleanup at midnight
	cleanUp := make(chan struct{}, 2)
	c := cron.New()
	_, err = c.AddFunc("@midnight", func() {
		cleanUp <- struct{}{}
	})
	if err != nil {
		return fmt.Errorf("failed to setup cron task: %w", err)
	}
	defer c.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("provision engine context done, exiting")
			return nil

		case reservation, ok := <-cReservation:
			if !ok {
				log.Info().Msg("reservation source is emptied. stopping engine")
				return nil
			}

			if reservation.last {
				isAllWorkloadsProcessed = true
				// Trigger cleanup by sending a struct onto the channel
				cleanUp <- struct{}{}
				continue
			}

			expired := reservation.Expired()
			slog := log.With().
				Str("id", string(reservation.ID)).
				Str("type", string(reservation.Type)).
				Str("duration", fmt.Sprintf("%v", reservation.Duration)).
				Str("tag", reservation.Tag.String()).
				Bool("to-delete", reservation.ToDelete).
				Bool("expired", expired).
				Logger()

			if expired || reservation.ToDelete {
				slog.Info().Msg("start decommissioning reservation")
				if err := e.decommission(ctx, &reservation.Reservation); err != nil {
					log.Error().Err(err).Msgf("failed to decommission reservation %s", reservation.ID)
					continue
				}

				log.Info().Msg("Replying successfull decomission to chain")
				if err := poller.submitExtrinsic(reservation.ID, "TemplateModule.contract_cancelled"); err != nil {
					log.Error().Err(err).Msgf("failed to reply for decommission of reservation %s", reservation.ID)
					continue
				}
				log.Info().Msg("Reply successfull")
			} else {
				slog.Info().Msg("start provisioning reservation")
				if err := e.provision(ctx, &reservation.Reservation); err != nil {
					log.Error().Err(err).Msgf("failed to provision reservation %s", reservation.ID)
					continue
				}

				log.Info().Msg("Replying successfull provision to chain")
				if err := poller.submitExtrinsic(reservation.ID, "TemplateModule.contract_deployed"); err != nil {
					log.Error().Err(err).Msgf("failed to reply for provision of reservation %s", reservation.ID)
					continue
				}
				log.Info().Msg("Reply successfull")
			}

			if err := e.updateStats(); err != nil {
				log.Error().Err(err).Msg("failed to updated the capacity counters")
			}

		case <-cleanUp:
			if !isAllWorkloadsProcessed {
				// only allow cleanup triggered by the cron to run once
				// we are sure all the workloads from the cache/explorer have been processed
				log.Info().Msg("all workloads not yet processed, delay cleanup")
				continue
			}
			log.Info().Msg("start cleaning up resources")
			if err := CleanupResources(ctx, e.zbusCl); err != nil {
				log.Error().Err(err).Msg("failed to cleanup resources")
				continue
			}
		}
	}
}

func (e *Engine) provision(ctx context.Context, r *Reservation) error {
	if err := r.validate(); err != nil {
		return errors.Wrapf(err, "failed validation of reservation")
	}

	fn, ok := e.provisioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	if r.Reference != "" {
		if err := e.migrateToPool(ctx, r); err != nil {
			return err
		}
	}

	if cached, err := e.cache.Get(r.ID); err == nil {
		log.Info().Str("id", r.ID).Msg("reservation have already been processed")
		if cached.Result.IsNil() {
			// this is probably an older reservation that is cached BEFORE
			// we start caching the result along with the reservation
			// then we just need to return here.
			return nil
		}

		// otherwise, it's safe to resend the same result
		// back to the grid.
		if err := e.reply(ctx, &cached.Result); err != nil {
			log.Error().Err(err).Msg("failed to send result to BCDB")
		}

		return nil
	}

	// to ensure old reservation workload that are already running
	// keeps running as it is, we use the reference as new workload ID
	realID := r.ID
	if r.Reference != "" {
		r.ID = r.Reference
	}

	returned, provisionError := fn(ctx, r)
	if provisionError != nil {
		log.Error().
			Err(provisionError).
			Str("id", r.ID).
			Msgf("failed to apply provision")
	} else {
		log.Info().
			Str("result", fmt.Sprintf("%v", returned)).
			Msgf("workload deployed")
	}

	result, err := e.buildResult(realID, r.Type, provisionError, returned)
	if err != nil {
		return errors.Wrapf(err, "failed to build result object for reservation: %s", result.ID)
	}

	// we make sure we store the reservation in cache first before
	// returning the reply back to the grid, this is to make sure
	// if the reply failed for any reason, the node still doesn't
	// try to redeploy that reservation.
	r.ID = realID
	r.Result = *result
	if err := e.cache.Add(r); err != nil {
		return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
	}

	// if err := e.reply(ctx, result); err != nil {
	// 	log.Error().Err(err).Msg("failed to send result to BCDB")
	// }

	// we skip the counting.
	if provisionError != nil {
		return provisionError
	}

	// If an update occurs on the network we don't increment the counter
	if r.Type == "network_resource" {
		nr := pkg.NetResource{}
		if err := json.Unmarshal(r.Data, &nr); err != nil {
			return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
		}

		uniqueID := NetworkID(r.User, nr.Name)
		exists, err := e.cache.NetworkExists(string(uniqueID))
		if err != nil {
			return errors.Wrap(err, "failed to check if network exists")
		}
		if exists {
			return nil
		}
	}

	if err := e.statser.Increment(r); err != nil {
		log.Err(err).Str("reservation_id", r.ID).Msg("failed to increment workloads statistics")
	}

	return nil
}

func (e *Engine) decommission(ctx context.Context, r *Reservation) error {
	fn, ok := e.decomissioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	exists, err := e.cache.Exists(r.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to check if reservation %s exists in cache", r.ID)
	}

	if !exists {
		log.Info().Str("id", r.ID).Msg("reservation not provisioned, no need to decomission")
		if err := e.feedback.Deleted(e.nodeID, r.ID); err != nil {
			log.Error().Err(err).Str("id", r.ID).Msg("failed to mark reservation as deleted")
		}
		return nil
	}

	// to ensure old reservation can be deleted
	// we use the reference as workload ID
	realID := r.ID
	if r.Reference != "" {
		r.ID = r.Reference
	}

	err = fn(ctx, r)
	if err != nil {
		return errors.Wrap(err, "decommissioning of reservation failed")
	}

	r.ID = realID

	if err := e.cache.Remove(r.ID); err != nil {
		return errors.Wrapf(err, "failed to remove reservation %s from cache", r.ID)
	}

	if err := e.statser.Decrement(r); err != nil {
		log.Err(err).Str("reservation_id", r.ID).Msg("failed to decrement workloads statistics")
	}

	// if err := e.feedback.Deleted(e.nodeID, r.ID); err != nil {
	// 	return errors.Wrap(err, "failed to mark reservation as deleted")
	// }

	return nil
}

func (e *Engine) reply(ctx context.Context, result *Result) error {
	log.Debug().Str("id", result.ID).Msg("sending reply for reservation")

	if err := e.signResult(result); err != nil {
		return err
	}

	return e.feedback.Feedback(e.nodeID, result)
}

func (e *Engine) buildResult(id string, typ ReservationType, err error, info interface{}) (*Result, error) {
	result := &Result{
		Type:    typ,
		Created: time.Now(),
		ID:      id,
	}

	if err != nil {
		result.Error = err.Error()
		result.State = StateError
	} else {
		result.State = StateOk
	}

	br, err := json.Marshal(info)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	return result, nil
}

func (e *Engine) signResult(result *Result) error {

	b, err := result.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to convert the result to byte for signature")
	}

	sig, err := e.signer.Sign(b)
	if err != nil {
		return errors.Wrap(err, "failed to signed the result")
	}
	result.Signature = hex.EncodeToString(sig)

	return nil
}

func (e *Engine) updateStats() error {
	wl := e.statser.CurrentWorkloads()
	r := e.statser.CurrentUnits()

	log.Info().
		Uint16("network", wl.Network).
		Uint16("volume", wl.Volume).
		Uint16("zDBNamespace", wl.ZDBNamespace).
		Uint16("container", wl.Container).
		Uint16("k8sVM", wl.K8sVM).
		Uint16("proxy", wl.Proxy).
		Uint16("reverseProxy", wl.ReverseProxy).
		Uint16("subdomain", wl.Subdomain).
		Uint16("delegateDomain", wl.DelegateDomain).
		Uint64("cru", r.Cru).
		Float64("mru", r.Mru).
		Float64("hru", r.Hru).
		Float64("sru", r.Sru).
		Msgf("provision statistics")

	return e.feedback.UpdateStats(e.nodeID, wl, r)
}

// Counters is a zbus stream that sends statistics from the engine
func (e *Engine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
			}

			wls := e.statser.CurrentWorkloads()
			pc := pkg.ProvisionCounters{
				Container: int64(wls.Container),
				Network:   int64(wls.Network),
				ZDB:       int64(wls.ZDBNamespace),
				Volume:    int64(wls.Volume),
				VM:        int64(wls.K8sVM),
			}

			select {
			case <-ctx.Done():
			case ch <- pc:
			}
		}
	}()

	return ch
}

func (e *Engine) migrateToPool(ctx context.Context, r *Reservation) error {

	oldRes, err := e.cache.Get(r.Reference)
	if err == nil && oldRes.ID == r.Reference {
		// we have received a reservation that reference another one.
		// This is the sign user is trying to migrate his workloads to the new capacity pool system

		log.Info().Str("reference", r.Reference).Msg("reservation referencing another one")

		if string(oldRes.Type) != "network" { //we skip network cause its a PITA
			// first let make sure both are the same
			if !bytes.Equal(oldRes.Data, r.Data) {
				return fmt.Errorf("trying to upgrade workloads to new version. new workload content is different from the old one. upgrade refused")
			}
		}

		// remove the old one from the cache and store the new one
		log.Info().Msgf("migration: remove %v from cache", oldRes.ID)
		if err := e.cache.Remove(oldRes.ID); err != nil {
			return err
		}
		log.Info().Msgf("migration: add %v to cache", r.ID)
		if err := e.cache.Add(r); err != nil {
			return err
		}

		r.Result.ID = r.ID
		if err := e.signResult(&r.Result); err != nil {
			return errors.Wrap(err, "error while signing reservation result")
		}

		if err := e.feedback.Feedback(e.nodeID, &r.Result); err != nil {
			return err
		}

		log.Info().Str("old_id", oldRes.ID).Str("new_id", r.ID).Msg("reservation upgraded to new system")
	}

	return nil
}

// NetworkID construct a network ID based on a userID and network name
func NetworkID(userID, name string) pkg.NetID {
	buf := bytes.Buffer{}
	buf.WriteString(userID)
	buf.WriteString(name)
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return pkg.NetID(string(b))
}

// SubstratePoller is a poller
type SubstratePoller struct {
	api      *gsrpc.SubstrateAPI
	identity *stubs.IdentityManagerStub
}

type contract struct {
	ResourcePrices resourcePrice
	AccountID      types.AccountID
	NodeID         []types.U8
	FarmerAccount  types.AccountID
	UserAccount    types.AccountID
	Accepted       bool
	// WorkloadState  []types.U8
	// ExpiresAt      types.U64
	// LastClaimed    types.U64
}

type resourcePrice struct {
	Currency types.U64
	Sru      types.U64
	Hru      types.U64
	Cru      types.U64
	Nru      types.U64
	Mru      types.U64
}

type Volume struct {
	DiskType types.U8
	Size     types.U64
}

type ContractAdded struct {
	Phase      types.Phase
	Who        types.AccountID
	NodeID     []types.U8
	ContractID types.U64
	Topics     []types.Hash
}

type ContractUpdated struct {
	Phase      types.Phase
	Who        types.AccountID
	ContractID types.U64
	Topics     []types.Hash
}

type ContractAccepted struct {
	Phase      types.Phase
	NodeID     []types.U8
	ContractID types.U64
	Topics     []types.Hash
}

type ContractPaid struct {
	Phase      types.Phase
	Who        types.AccountID
	ContractID types.U64
	Topics     []types.Hash
}

type ContractDeployed struct {
	Phase      types.Phase
	NodeID     []types.U8
	ContractID types.U64
	Topics     []types.Hash
}

type ContractCancelled struct {
	Phase      types.Phase
	NodeID     []types.U8
	ContractID types.U64
	Topics     []types.Hash
}

type EventRecords struct {
	types.EventRecords
	TemplateModule_ContractAdded     []ContractAdded
	TemplateModule_ContractPaid      []ContractPaid
	TemplateModule_ContractUpdated   []ContractUpdated
	TemplateModule_ContractCancelled []ContractCancelled
	TemplateModule_ContractDeployed  []ContractDeployed
	TemplateModule_ContractAccepted  []ContractAccepted
}

// NewSubstratePoller creates a poller
func NewSubstratePoller(identity *stubs.IdentityManagerStub) *SubstratePoller {
	api, err := gsrpc.NewSubstrateAPI("ws://192.168.0.170:9944")
	if err != nil {
		log.Err(err).Msg("failed to create substrate api client")
		return nil
	}

	return &SubstratePoller{
		api:      api,
		identity: identity,
	}
}

// GetReservations implements provision.ReservationPoller
func (s *SubstratePoller) GetReservations(ctx context.Context) <-chan *ReservationJob {
	ch := make(chan *ReservationJob)

	meta, err := s.api.RPC.State.GetMetadataLatest()
	if err != nil {
		panic(err)
	}

	key, err := types.CreateStorageKey(meta, "System", "Events", nil, nil)
	if err != nil {
		panic(err)
	}

	go func() {
		sub, err := s.api.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
		if err != nil {
			panic(err)
		}
		defer sub.Unsubscribe()
		// outer for loop for subscription notifications
		for {
			set := <-sub.Chan()
			// inner loop for the changes within one of those notifications
			for _, chng := range set.Changes {
				if !types.Eq(chng.StorageKey, key) || !chng.HasStorageData {
					// skip, we are only interested in events with content
					continue
				}

				// Decode the event records
				events := EventRecords{}
				err = types.EventRecordsRaw(chng.StorageData).DecodeEventRecords(meta, &events)
				if err != nil {
					panic(err)
				}

				for _, e := range events.TemplateModule_ContractPaid {
					log.Info().Msg("Contract paid event received, parsing now")
					res, err := s.parseContract(e.ContractID, false)
					if err != nil {
						log.Err(err).Msg("Error parsing contract")
						continue
					}

					// If reservation ID is empty this means this reservation is not for this node to deploy
					if res.ID == "" {
						continue
					}

					reservation := ReservationJob{
						res,
						false,
					}
					ch <- &reservation
				}
				for _, e := range events.TemplateModule_ContractCancelled {
					log.Info().Msg("Contract cancelled event received, parsing now")
					res, err := s.parseContract(e.ContractID, true)
					if err != nil {
						log.Err(err).Msg("Error parsing contract")
						continue
					}

					// If reservation ID is empty this means this reservation is not for this node to deploy
					if res.ID == "" {
						continue
					}

					reservation := ReservationJob{
						res,
						false,
					}
					ch <- &reservation
				}
			}
		}
	}()

	return ch
}

// Volume defines a mount point
type volume struct {
	// Size of the volume in GiB
	Size uint64 `json:"size"`
	// Type of disk underneath the volume
	Type pkg.DeviceType `json:"type"`
}

func (s *SubstratePoller) getContract(contractID types.U64) (contract, error) {
	log.Info().Msgf("Fetching contract %+v", contractID)
	meta, err := s.api.RPC.State.GetMetadataLatest()
	if err != nil {
		return contract{}, err
	}

	buf := bytes.NewBuffer(nil)
	enc := scale.NewEncoder(buf)
	if err := enc.Encode(contractID); err != nil {
		return contract{}, err
	}
	k := buf.Bytes()

	key, err := types.CreateStorageKey(meta, "TemplateModule", "Contracts", k, nil)
	if err != nil {
		return contract{}, err
	}

	var c contract
	ok, err := s.api.RPC.State.GetStorageLatest(key, &c)
	if err != nil || !ok {
		log.Info().Msgf("Failed to decode contract with id %+v", contractID)
		return contract{}, err
	}

	return c, nil
}

func byteSliceToString(bs []types.U8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

func (s *SubstratePoller) parseContract(contractID types.U64, toDelete bool) (Reservation, error) {
	meta, err := s.api.RPC.State.GetMetadataLatest()
	if err != nil {
		return Reservation{}, err
	}

	contract, err := s.getContract(contractID)
	if err != nil {
		return Reservation{}, nil
	}

	nodeID := byteSliceToString(contract.NodeID)
	// If the nodeID does not equal this node's ID we return an empty reservation struct
	// This will tell provisiond to skip this item
	if nodeID != string(s.identity.NodeID()) {
		return Reservation{}, nil
	}

	buf := bytes.NewBuffer(nil)
	enc := scale.NewEncoder(buf)
	if err := enc.Encode(contractID); err != nil {
		return Reservation{}, err
	}
	k := buf.Bytes()

	key, err := types.CreateStorageKey(meta, "TemplateModule", "VolumeReservations", k, nil)
	if err != nil {
		return Reservation{}, err
	}

	var v Volume
	ok, err := s.api.RPC.State.GetStorageLatest(key, &v)
	if err != nil || !ok {
		log.Info().Msgf("Failed to decode volume with id %+v", contractID)
		return Reservation{}, err
	}

	var reservation Reservation

	volume := volume{
		Size: uint64(v.Size),
	}
	switch v.DiskType {
	case 1:
		volume.Type = pkg.HDDDevice
	case 2:
		volume.Type = pkg.SSDDevice
	default:

	}

	reservation.Data, err = json.Marshal(volume)
	if err != nil {
		return reservation, err
	}

	reservation.ToDelete = toDelete
	reservation.Type = "volume"
	reservation.Duration = time.Hour * 2
	reservation.Created = time.Now()
	ID := strconv.Itoa(int(contractID))
	reservation.ID = string(ID)

	return reservation, nil
}

func (s *SubstratePoller) submitExtrinsic(reservationID, call string) error {
	meta, err := s.api.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(reservationID)
	if err != nil {
		return err
	}

	c, err := types.NewCall(meta, "TemplateModule.contract_deployed", types.U64(id))
	if err != nil {
		return err
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := s.api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return err
	}

	rv, err := s.api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return err
	}

	str := types.HexEncodeToString(s.identity.PrivateKey()[:32])

	identity, err := signature.KeyringPairFromSecret(str, "")
	log.Info().Msgf("Node address: %s", identity.Address)

	key, err := types.CreateStorageKey(meta, "System", "Account", identity.PublicKey, nil)
	if err != nil {
		return err
	}

	var accountInfo types.AccountInfo
	ok, err := s.api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		if !ok {
			return fmt.Errorf("error getting account info")
		}
		return err
	}

	nonce := uint32(accountInfo.Nonce)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: 1,
	}

	err = ext.Sign(identity, o)
	if err != nil {
		return err
	}

	// Send the extrinsic
	_, err = s.api.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return err
	}

	return nil
}

func (s *SubstratePoller) createAddressFromID() error {
	str := types.HexEncodeToString(s.identity.PrivateKey()[:32])
	identity, err := signature.KeyringPairFromSecret(str, "")
	if err != nil {
		return err
	}
	log.Info().Msgf("Node address: %s", identity.Address)

	return nil
}
