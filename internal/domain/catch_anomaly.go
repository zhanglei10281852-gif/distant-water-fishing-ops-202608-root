package domain

import (
	"strings"
	"time"
)

type CatchAnomalyStatus string

const (
	CatchAnomalyOpen      CatchAnomalyStatus = "open"
	CatchAnomalyReviewing CatchAnomalyStatus = "reviewing"
	CatchAnomalyCleared   CatchAnomalyStatus = "cleared"
	CatchAnomalyRejected  CatchAnomalyStatus = "rejected"
)

type CatchDeclaration struct {
	ID              string            `json:"id"`
	FishingVoyageID string            `json:"fishing_voyage_id"`
	SpeciesCode     string            `json:"species_code"`
	Sequence        int64             `json:"sequence"`
	CatchVariance   GramCatchVariance `json:"catch_variance_grams"`
	RecordedAt      time.Time         `json:"recorded_at"`
	ReceivedAt      time.Time         `json:"received_at"`
}

type CatchAnomaly struct {
	ID                 string             `json:"id"`
	FishingVoyageID    string             `json:"fishing_voyage_id"`
	Status             CatchAnomalyStatus `json:"status"`
	FirstDeclarationAt time.Time          `json:"first_declaration_at"`
	LastDeclarationAt  time.Time          `json:"last_declaration_at"`
	Minimum            GramCatchVariance  `json:"minimum_catch_variance_grams"`
	Maximum            GramCatchVariance  `json:"maximum_catch_variance_grams"`
	DeclarationCount   int                `json:"declaration_count"`
	ReviewDueAt        time.Time          `json:"review_due_at"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	Version            int64              `json:"version"`
}

type AnomalyDisposition struct {
	ID             string
	CatchAnomalyID string
	Reviewer       string
	Decision       CatchAnomalyStatus
	Rationale      string
	CreatedAt      time.Time
}

func (s CatchAnomalyStatus) IsResolved() bool {
	return s == CatchAnomalyCleared || s == CatchAnomalyRejected
}

func (s CatchAnomalyStatus) IsOpen() bool {
	return s == CatchAnomalyOpen || s == CatchAnomalyReviewing
}

func (r CatchDeclaration) Validate() error {
	if strings.TrimSpace(r.FishingVoyageID) == "" || strings.TrimSpace(r.SpeciesCode) == "" {
		return FieldError{Field: "catch_declaration", Message: "fishing voyage and species code are required"}
	}
	if r.Sequence < 1 || r.RecordedAt.IsZero() {
		return FieldError{Field: "declaration", Message: "sequence and recorded_at are required"}
	}
	return nil
}

func (e *CatchAnomaly) Include(declaration CatchDeclaration, now time.Time) {
	if e.DeclarationCount == 0 || declaration.RecordedAt.Before(e.FirstDeclarationAt) {
		e.FirstDeclarationAt = declaration.RecordedAt
	}
	if e.DeclarationCount == 0 || declaration.RecordedAt.After(e.LastDeclarationAt) {
		e.LastDeclarationAt = declaration.RecordedAt
	}
	if e.DeclarationCount == 0 || declaration.CatchVariance < e.Minimum {
		e.Minimum = declaration.CatchVariance
	}
	if e.DeclarationCount == 0 || declaration.CatchVariance > e.Maximum {
		e.Maximum = declaration.CatchVariance
	}
	e.DeclarationCount++
	e.UpdatedAt = now.UTC()
}

func (e *CatchAnomaly) StartReview(now time.Time) error {
	if e.Status != CatchAnomalyOpen {
		return TransitionError{Entity: "catch_anomaly", From: string(e.Status), To: string(CatchAnomalyReviewing)}
	}
	e.Status = CatchAnomalyReviewing
	e.UpdatedAt = now.UTC()
	return nil
}

func (e *CatchAnomaly) Decide(decision CatchAnomalyStatus, now time.Time) error {
	if e.Status != CatchAnomalyOpen && e.Status != CatchAnomalyReviewing {
		return TransitionError{Entity: "catch_anomaly", From: string(e.Status), To: string(decision)}
	}
	if decision != CatchAnomalyCleared && decision != CatchAnomalyRejected {
		return FieldError{Field: "decision", Message: "must be cleared or rejected"}
	}
	e.Status = decision
	e.UpdatedAt = now.UTC()
	return nil
}
