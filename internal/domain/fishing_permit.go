package domain

import (
	"strings"
	"time"
)

type FishingPermitStatus string

const (
	FishingPermitDraft    FishingPermitStatus = "draft"
	FishingPermitActive   FishingPermitStatus = "active"
	FishingPermitArchived FishingPermitStatus = "archived"
)

type FishingPermit struct {
	ID                       string                `json:"id"`
	Code                     string                `json:"code"`
	Name                     string                `json:"name"`
	Status                   FishingPermitStatus   `json:"status"`
	CatchVariance            CatchVarianceEnvelope `json:"catch_variance"`
	MaxVoyageDuration        time.Duration         `json:"max_voyage_duration"`
	ComplianceReviewDeadline time.Duration         `json:"compliance_review_deadline"`
	BusinessTimezone         string                `json:"business_timezone"`
	CreatedAt                time.Time             `json:"created_at"`
	UpdatedAt                time.Time             `json:"updated_at"`
	Version                  int64                 `json:"version"`
}

func (s FishingPermit) Validate() error {
	if strings.TrimSpace(s.Code) == "" {
		return FieldError{Field: "code", Message: "is required"}
	}
	if strings.TrimSpace(s.Name) == "" {
		return FieldError{Field: "name", Message: "is required"}
	}
	if err := s.CatchVariance.Validate(); err != nil {
		return err
	}
	if s.MaxVoyageDuration <= 0 || s.MaxVoyageDuration > 14*24*time.Hour {
		return FieldError{Field: "max_voyage_duration", Message: "must be between zero and fourteen days"}
	}
	if s.ComplianceReviewDeadline <= 0 || s.ComplianceReviewDeadline > 7*24*time.Hour {
		return FieldError{Field: "compliance_review_deadline", Message: "must be between zero and seven days"}
	}
	if _, err := time.LoadLocation(s.BusinessTimezone); err != nil {
		return FieldError{Field: "business_timezone", Message: "is invalid"}
	}
	switch s.Status {
	case FishingPermitDraft, FishingPermitActive, FishingPermitArchived:
		return nil
	default:
		return FieldError{Field: "status", Message: "is invalid"}
	}
}

func (s FishingPermit) CanAcceptFishingVoyages() bool { return s.Status == FishingPermitActive }

func (s FishingPermit) VoyageWithinLimit(startAt, finishAt time.Time) bool {
	return finishAt.After(startAt) && finishAt.Sub(startAt) <= s.MaxVoyageDuration
}

func (s FishingPermit) IsClosed() bool { return s.Status == FishingPermitArchived }
