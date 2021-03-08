package gridtypes

import (
	"fmt"
	"io"
)

// https://github.com/threefoldtech/vgrid/blob/main/zosreq/req_deployment_root.v

// DeploymentHeader contains all deployment related information
// it's split in a separate object so it can stored and processed
// separately unrelated to managed workloads
type DeploymentHeader struct {
	Version              int                  `json:"version"`
	TwinID               uint32               `json:"twin_id"`
	DeploymentID         uint32               `json:"deployment_id"`
	Metadata             string               `json:"metadata"`
	Description          string               `json:"description"`
	Expiration           Timestamp            `json:"expiration"`
	SignatureRequirement SignatureRequirement `json:"signature_requirement"`
}

// Challenge compute signature challenge for the header
func (h *DeploymentHeader) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", h.Version); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", h.TwinID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", h.DeploymentID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", h.Metadata); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", h.Description); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", h.Expiration); err != nil {
		return err
	}

	return nil
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

// Deployment is a deployment envelope
type Deployment struct {
	DeploymentHeader
	Workloads []Workload `json:"workloads"`
}

// Challenge computes challenge for SignatureRequest
func (d *Deployment) Challenge(w io.Writer) error {
	if err := d.DeploymentHeader.Challenge(w); err != nil {
		return err
	}

	for _, wl := range d.Workloads {
		if err := wl.Challenge(w); err != nil {
			return err
		}
	}

	return nil
}
