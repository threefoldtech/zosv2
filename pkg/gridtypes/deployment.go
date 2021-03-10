package gridtypes

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrWorkloadNotFound error
	ErrWorkloadNotFound = fmt.Errorf("workload not found")
)

// https://github.com/threefoldtech/vgrid/blob/main/zosreq/req_deployment_root.v

// Deployment structure
type Deployment struct {
	Version              int                  `json:"version"`
	TwinID               uint32               `json:"twin_id"`
	DeploymentID         uint32               `json:"deployment_id"`
	Metadata             string               `json:"metadata"`
	Description          string               `json:"description"`
	Expiration           Timestamp            `json:"expiration"`
	SignatureRequirement SignatureRequirement `json:"signature_requirement"`
	Workloads            []Workload           `json:"workloads"`
}

// WorkloadWithID wrapper around workload type
// that holds the global workload ID
type WorkloadWithID struct {
	*Workload
	ID WorkloadID
}

// SignatureRequest struct
type SignatureRequest struct {
	TwinID   uint32 `json:"twin_id"`
	Required bool   `json:"required"`
	Weight   uint   `json:"weight"`
}

// Challenge computes challenge for SignatureRequest
func (r *SignatureRequest) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", r.TwinID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%t", r.Required); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", r.Weight); err != nil {
		return err
	}

	return nil
}

// Signature struct
type Signature struct {
	TwinID    uint32 `json:"twin_id"`
	Signature string `json:"signature"`
}

// SignatureRequirement struct
type SignatureRequirement struct {
	Requests       []SignatureRequest `json:"requests"`
	WeightRequired int                `json:"weight_required"`
	Signatures     []Signature        `json:"signatures"`
}

// Challenge computes challenge for SignatureRequest
func (r *SignatureRequirement) Challenge(w io.Writer) error {
	for _, request := range r.Requests {
		if err := request.Challenge(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%d", r.WeightRequired); err != nil {
		return err
	}

	return nil
}

// Challenge computes challenge for SignatureRequest
func (d *Deployment) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", d.Version); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.TwinID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.DeploymentID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", d.Metadata); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", d.Description); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.Expiration); err != nil {
		return err
	}

	for _, wl := range d.Workloads {
		if err := wl.Challenge(w); err != nil {
			return err
		}
	}

	return nil
}

// Valid validates deployment structure
func (d *Deployment) Valid() error {
	names := make(map[string]struct{})
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		name := wl.Name
		if err := IsValidName(name); err != nil {
			return errors.Wrapf(err, "name '%s' is invalid", name)
		}

		if _, ok := names[name]; ok {
			return fmt.Errorf("multiple workloads with the same name '%s'", name)
		}
		names[name] = struct{}{}

		if err := wl.Valid(d); err != nil {
			return err
		}
	}

	return nil
}

// Get a workload by name
func (d *Deployment) Get(name string) (*WorkloadWithID, error) {
	if err := IsValidName(name); err != nil {
		return nil, err
	}

	for i := range d.Workloads {
		wl := &d.Workloads[i]
		id, _ := NewWorkloadID(d.TwinID, d.DeploymentID, name)
		if wl.Name == name {
			return &WorkloadWithID{
				Workload: wl,
				ID:       id,
			}, nil
		}
	}

	return nil, errors.Wrapf(ErrWorkloadNotFound, "no workload with name '%s'", name)
}

// GetType gets a reservation by name if only of the correct type.
func (d *Deployment) GetType(name string, typ WorkloadType) (*WorkloadWithID, error) {
	wl, err := d.Get(name)
	if err != nil {
		return nil, err
	}
	if wl.Type != typ {
		return nil, fmt.Errorf("workload '%s' of wrong type", name)
	}

	return wl, nil
}

// ByType gets all workloads from this reservation by type.
func (d *Deployment) ByType(typ WorkloadType) []*WorkloadWithID {
	var results []*WorkloadWithID
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		if wl.Type == typ {
			id, err := NewWorkloadID(d.TwinID, d.DeploymentID, wl.Name)
			if err != nil {
				log.Warn().Err(err).Msg("deployment has invalid name. please run validation")
				continue
			}

			results = append(
				results,
				&WorkloadWithID{
					Workload: wl,
					ID:       id,
				},
			)
		}
	}

	return results
}
