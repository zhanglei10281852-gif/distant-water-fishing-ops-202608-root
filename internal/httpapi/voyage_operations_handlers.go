package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/service"
)

type createCatchLandingTaskRequest struct {
	CoordinatorID      string `json:"coordinator_id"`
	FisheriesOfficerID string `json:"fisheries_officer_id"`
	LandingStation     string `json:"landing_station"`
}

func (s *Server) createCatchLandingTask(w http.ResponseWriter, r *http.Request) {
	var input createCatchLandingTaskRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	catch_landing_task, err := s.services.Handoffs.CreateCatchLandingTask(r.Context(), service.CreateCatchLandingTaskInput{FishingVoyageID: parseID(r), CoordinatorID: input.CoordinatorID, FisheriesOfficerID: input.FisheriesOfficerID, LandingStation: input.LandingStation})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, catch_landing_task)
}

type resolveCatchLandingTaskRequest struct {
	Accepted bool   `json:"accepted"`
	Note     string `json:"note"`
}

func (s *Server) resolveCatchLandingTask(w http.ResponseWriter, r *http.Request) {
	var input resolveCatchLandingTaskRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	catch_landing_task, err := s.services.Handoffs.ResolveCatchLandingTask(r.Context(), parseID(r), input.Accepted, input.Note)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, catch_landing_task)
}

type declarationRequest struct {
	SpeciesCode        string    `json:"species_code"`
	Sequence           int64     `json:"sequence"`
	CatchVarianceValue float64   `json:"catch_variance"`
	RecordedAt         time.Time `json:"recorded_at"`
}

func (s *Server) submitCatchDeclaration(w http.ResponseWriter, r *http.Request) {
	var input declarationRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	catch_variance, err := domain.CatchVarianceFromKilograms(input.CatchVarianceValue)
	if err != nil {
		writeError(w, r, err)
		return
	}
	declaration, catch_anomaly, err := s.services.CatchDeclaration.SubmitCatchDeclaration(r.Context(), service.SubmitCatchDeclarationInput{FishingVoyageID: parseID(r), SpeciesCode: input.SpeciesCode, Sequence: input.Sequence, CatchVariance: catch_variance, RecordedAt: input.RecordedAt})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"declaration": declaration, "catch_anomaly": catch_anomaly})
}

func (s *Server) listLandingAnomalies(w http.ResponseWriter, r *http.Request) {
	dueBefore, err := parseTimeQuery(r.URL.Query().Get("due_before"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	page, err := s.services.Query.LandingAnomalies(r.Context(), repository.CatchAnomalyFilter{Page: parsePage(r), FishingVoyageID: r.URL.Query().Get("fishing_voyage_id"), Status: domain.CatchAnomalyStatus(r.URL.Query().Get("status")), DueBefore: dueBefore})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) startReview(w http.ResponseWriter, r *http.Request) {
	catch_anomaly, err := s.services.Review.StartReview(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, catch_anomaly)
}

type decisionRequest struct {
	Decision  domain.CatchAnomalyStatus `json:"decision"`
	Rationale string                    `json:"rationale"`
}

func (s *Server) decideCatchAnomaly(w http.ResponseWriter, r *http.Request) {
	var input decisionRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	catch_anomaly, err := s.services.Review.Decide(r.Context(), service.DecideInput{CatchAnomalyID: parseID(r), Decision: input.Decision, Rationale: input.Rationale})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, catch_anomaly)
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	page, err := s.services.Query.Audit(r.Context(), repository.AuditFilter{Page: parsePage(r), EntityType: r.URL.Query().Get("entity_type"), EntityID: r.URL.Query().Get("entity_id"), Actor: r.URL.Query().Get("actor"), RequestID: r.URL.Query().Get("request_id")})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.services.Query.PlatformSummary(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func queryInt(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}
