package httpapi

import (
	"net/http"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
)

type bulkRegisterRequest struct {
	FishingVessels []registerFishingVesselRequest `json:"fishing_vessels"`
}

func (s *Server) bulkRegisterFishingVessels(w http.ResponseWriter, r *http.Request) {
	var input bulkRegisterRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	stages := make([]domain.FishingVessel, 0, len(input.FishingVessels))
	for _, item := range input.FishingVessels {
		stages = append(stages, domain.FishingVessel{FishingPermitID: item.FishingPermitID, DeparturePortID: item.DeparturePortID, RegistryNumber: item.RegistryNumber, VesselClass: item.VesselClass, VoyageCount: item.VoyageCount, HoldCapacityKg: item.HoldCapacityKg, ExpiresAt: item.ExpiresAt})
	}
	result, err := s.services.Catalog.BulkRegisterFishingVessels(r.Context(), stages)
	if err != nil {
		writeError(w, r, err)
		return
	}
	status := http.StatusCreated
	if result.Failed > 0 {
		status = http.StatusMultiStatus
	}
	writeJSON(w, status, result)
}

func (s *Server) startSupportFleetMaintenance(w http.ResponseWriter, r *http.Request) {
	support_fleet, err := s.services.SupportFleets.StartMaintenance(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, support_fleet)
}

func (s *Server) completeSupportFleetMaintenance(w http.ResponseWriter, r *http.Request) {
	support_fleet, err := s.services.SupportFleets.CompleteMaintenance(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, support_fleet)
}

func (s *Server) retireSupportFleet(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	support_fleet, err := s.services.SupportFleets.Retire(r.Context(), parseID(r), input.Reason)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, support_fleet)
}
