package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/service"
)

type planFishingVoyageRequest struct {
	FishingPermitID        string    `json:"fishing_permit_id"`
	DeparturePortID        string    `json:"departure_port_id"`
	LandingPortID          string    `json:"landing_port_id"`
	SupportFleetID         string    `json:"support_fleet_id"`
	VoyageCode             string    `json:"voyage_code"`
	FishingVesselIDs       []string  `json:"fishing_vessel_ids"`
	DepartureWindowOpensAt time.Time `json:"departure_window_opens_at"`
	LandingDeadlineAt      time.Time `json:"landing_deadline_at"`
}

func (s *Server) planFishingVoyage(w http.ResponseWriter, r *http.Request) {
	var input planFishingVoyageRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	run, err := s.services.FishingVoyages.PlanFishingVoyage(r.Context(), service.PlanFishingVoyageInput{FishingPermitID: input.FishingPermitID, DeparturePortID: input.DeparturePortID, LandingPortID: input.LandingPortID, SupportFleetID: input.SupportFleetID, VoyageCode: input.VoyageCode, FishingVesselIDs: append([]string(nil), input.FishingVesselIDs...), DepartureWindowOpensAt: input.DepartureWindowOpensAt, LandingDeadlineAt: input.LandingDeadlineAt, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) getFishingVoyage(w http.ResponseWriter, r *http.Request) {
	run, items, err := s.services.Query.FishingVoyage(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fishing_voyage": run, "fishing_vessels": items})
}

func (s *Server) getFishingVoyageReadiness(w http.ResponseWriter, r *http.Request) {
	report, err := s.services.Query.GetFishingVoyageReadiness(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) listFishingVoyages(w http.ResponseWriter, r *http.Request) {
	from, err := parseTimeQuery(r.URL.Query().Get("from"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r.URL.Query().Get("to"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	page, err := s.services.Query.FishingVoyages(r.Context(), repository.FishingVoyageFilter{Page: parsePage(r), FishingPermitID: r.URL.Query().Get("fishing_permit_id"), DeparturePortID: r.URL.Query().Get("departure_port_id"), LandingPortID: r.URL.Query().Get("landing_port_id"), State: domain.FishingVoyageState(r.URL.Query().Get("state")), From: from, To: to})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) clearFishingVoyage(w http.ResponseWriter, r *http.Request) {
	s.writeFishingVoyageTransition(w, r, s.services.FishingVoyages.ClearFishingVoyage)
}
func (s *Server) departureFishingVoyage(w http.ResponseWriter, r *http.Request) {
	s.writeFishingVoyageTransition(w, r, s.services.FishingVoyages.DepartureFishingVoyage)
}
func (s *Server) confirmFishingVoyageLanding(w http.ResponseWriter, r *http.Request) {
	s.writeFishingVoyageTransition(w, r, s.services.FishingVoyages.ConfirmFishingVoyageLanding)
}
func (s *Server) closeFishingVoyage(w http.ResponseWriter, r *http.Request) {
	s.writeFishingVoyageTransition(w, r, s.services.FishingVoyages.CloseFishingVoyage)
}

func (s *Server) writeFishingVoyageTransition(w http.ResponseWriter, r *http.Request, transition func(context.Context, string) (domain.FishingVoyage, error)) {
	run, err := transition(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) cancelFishingVoyage(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	run, err := s.services.FishingVoyages.CancelFishingVoyage(r.Context(), parseID(r), input.Note)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}
