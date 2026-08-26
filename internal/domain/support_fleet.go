package domain

import (
	"strings"
	"time"
)

type SupportFleetState string

const (
	SupportFleetStandby     SupportFleetState = "standby"
	SupportFleetAssigned    SupportFleetState = "assigned"
	SupportFleetDeployed    SupportFleetState = "deployed"
	SupportFleetMaintenance SupportFleetState = "maintenance"
	SupportFleetRetired     SupportFleetState = "retired"
)

type SupportFleet struct {
	ID                 string            `json:"id"`
	FleetCode          string            `json:"fleet_code"`
	State              SupportFleetState `json:"state"`
	CargoCapacityKg    int               `json:"cargo_capacity_kg"`
	CertificationDueAt time.Time         `json:"certification_due_at"`
	LastInspectionAt   time.Time         `json:"last_inspected_at"`
	AssignedVoyageID   string            `json:"assigned_voyage_id,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Version            int64             `json:"version"`
}

func (c SupportFleet) Validate() error {
	if strings.TrimSpace(c.FleetCode) == "" {
		return FieldError{Field: "fleet_code", Message: "is required"}
	}
	if c.CargoCapacityKg < 100 || c.CargoCapacityKg > 1000000 {
		return FieldError{Field: "cargo_capacity_kg", Message: "outside supported range"}
	}
	if c.CertificationDueAt.IsZero() || c.LastInspectionAt.IsZero() {
		return FieldError{Field: "support_fleet", Message: "certification and maintenance timestamps are required"}
	}
	switch c.State {
	case SupportFleetStandby, SupportFleetAssigned, SupportFleetDeployed, SupportFleetMaintenance, SupportFleetRetired:
		return nil
	default:
		return FieldError{Field: "support_fleet_state", Message: "is invalid"}
	}
}

func (c SupportFleet) EligibleFor(plannedStart time.Time, volume int) error {
	if c.State != SupportFleetStandby {
		return ConflictError{Resource: "support_fleet", Reason: "not on standby"}
	}
	if !c.IsCertifiedAt(plannedStart) {
		return ConflictError{Resource: "support_fleet", Reason: "certification expires before departure window opening"}
	}
	if c.CargoCapacityKg < volume {
		return ErrCapacityExceeded
	}
	return nil
}

func (c SupportFleet) IsCertifiedAt(at time.Time) bool {
	return c.CertificationDueAt.After(at) && !c.LastInspectionAt.After(at)
}

func (c SupportFleet) NeedsMaintenance(at time.Time) bool {
	return c.LastInspectionAt.IsZero() || c.LastInspectionAt.Before(at.Add(-72*time.Hour))
}

func (c *SupportFleet) StartMaintenance(now time.Time) error {
	if c.State != SupportFleetStandby {
		return TransitionError{Entity: "support_fleet", From: string(c.State), To: string(SupportFleetMaintenance)}
	}
	c.State = SupportFleetMaintenance
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *SupportFleet) CompleteMaintenance(now time.Time) error {
	if c.State != SupportFleetMaintenance {
		return TransitionError{Entity: "support_fleet", From: string(c.State), To: string(SupportFleetStandby)}
	}
	c.State = SupportFleetStandby
	c.LastInspectionAt = now.UTC()
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *SupportFleet) Retire(now time.Time) error {
	if c.State != SupportFleetStandby && c.State != SupportFleetMaintenance {
		return ConflictError{Resource: "support_fleet", Reason: "active voyage reservation must be released before retirement"}
	}
	c.State = SupportFleetRetired
	c.AssignedVoyageID = ""
	c.UpdatedAt = now.UTC()
	return nil
}
