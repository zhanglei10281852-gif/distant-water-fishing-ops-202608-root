package httpapi

import (
	"net/http"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
)

type createFishingPermitRequest struct {
	Code                   string  `json:"code"`
	Name                   string  `json:"name"`
	MinimumCatchVariance   float64 `json:"minimum_catch_variance_kg"`
	MaximumCatchVariance   float64 `json:"maximum_catch_variance_kg"`
	MaxVoyageDurationHours int     `json:"max_voyage_duration_hours"`
	ReviewHours            int     `json:"review_hours"`
	BusinessTimezone       string  `json:"business_timezone"`
}

func (s *Server) createFishingPermit(w http.ResponseWriter, r *http.Request) {
	var input createFishingPermitRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	minimum, err := domain.CatchVarianceFromKilograms(input.MinimumCatchVariance)
	if err != nil {
		writeError(w, r, err)
		return
	}
	maximum, err := domain.CatchVarianceFromKilograms(input.MaximumCatchVariance)
	if err != nil {
		writeError(w, r, err)
		return
	}
	rangeValue, err := domain.NewCatchVarianceEnvelope(minimum, maximum)
	if err != nil {
		writeError(w, r, err)
		return
	}
	program, err := s.services.Catalog.CreateFishingPermit(r.Context(), domain.FishingPermit{Code: input.Code, Name: input.Name, CatchVariance: rangeValue, MaxVoyageDuration: time.Duration(input.MaxVoyageDurationHours) * time.Hour, ComplianceReviewDeadline: time.Duration(input.ReviewHours) * time.Hour, BusinessTimezone: input.BusinessTimezone})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, program)
}

func (s *Server) activateFishingPermit(w http.ResponseWriter, r *http.Request) {
	program, err := s.services.Catalog.ActivateFishingPermit(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, program)
}

type createPortFacilityRequest struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Timezone   string `json:"timezone"`
	DailyLimit int    `json:"daily_limit"`
	CutoffHour int    `json:"cutoff_hour"`
}

func (s *Server) createPortFacility(w http.ResponseWriter, r *http.Request) {
	var input createPortFacilityRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	port_facility, err := s.services.Catalog.CreatePortFacility(r.Context(), domain.PortFacility{Code: input.Code, Name: input.Name, Timezone: input.Timezone, DailyLimit: input.DailyLimit, CutoffHour: input.CutoffHour})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, port_facility)
}

type createSupportFleetRequest struct {
	FleetCode          string    `json:"fleet_code"`
	CargoCapacityKg    int       `json:"cargo_capacity_kg"`
	CertificationDueAt time.Time `json:"certification_due_at"`
	LastInspectionAt   time.Time `json:"last_inspected_at"`
}

func (s *Server) createSupportFleet(w http.ResponseWriter, r *http.Request) {
	var input createSupportFleetRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	support_fleet, err := s.services.Catalog.CreateSupportFleet(r.Context(), domain.SupportFleet{FleetCode: input.FleetCode, CargoCapacityKg: input.CargoCapacityKg, CertificationDueAt: input.CertificationDueAt, LastInspectionAt: input.LastInspectionAt})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, support_fleet)
}

type registerFishingVesselRequest struct {
	FishingPermitID string    `json:"fishing_permit_id"`
	DeparturePortID string    `json:"departure_port_id"`
	RegistryNumber  string    `json:"registry_number"`
	VesselClass     string    `json:"vessel_class"`
	VoyageCount     int       `json:"voyage_count"`
	HoldCapacityKg  int       `json:"hold_capacity_kg"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func (s *Server) registerFishingVessel(w http.ResponseWriter, r *http.Request) {
	var input registerFishingVesselRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	batch, err := s.services.Catalog.RegisterFishingVessel(r.Context(), domain.FishingVessel{FishingPermitID: input.FishingPermitID, DeparturePortID: input.DeparturePortID, RegistryNumber: input.RegistryNumber, VesselClass: input.VesselClass, VoyageCount: input.VoyageCount, HoldCapacityKg: input.HoldCapacityKg, ExpiresAt: input.ExpiresAt})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, batch)
}

func (s *Server) verifyFishingVessel(w http.ResponseWriter, r *http.Request) {
	batch, err := s.services.Catalog.VerifyFishingVessel(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}
