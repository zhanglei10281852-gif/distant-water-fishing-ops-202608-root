package domain

import (
	"fmt"
	"strings"
	"time"
)

type PortFacilityStatus string

const (
	PortFacilityActive    PortFacilityStatus = "active"
	PortFacilitySuspended PortFacilityStatus = "suspended"
)

type PortFacility struct {
	ID         string             `json:"id"`
	Code       string             `json:"code"`
	Name       string             `json:"name"`
	Timezone   string             `json:"timezone"`
	Status     PortFacilityStatus `json:"status"`
	DailyLimit int                `json:"daily_limit"`
	CutoffHour int                `json:"cutoff_hour"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Version    int64              `json:"version"`
}

func (s PortFacility) Validate() error {
	if strings.TrimSpace(s.Code) == "" || strings.TrimSpace(s.Name) == "" {
		return FieldError{Field: "port_facility", Message: "code and name are required"}
	}
	if _, err := time.LoadLocation(s.Timezone); err != nil {
		return FieldError{Field: "timezone", Message: "is invalid"}
	}
	if s.DailyLimit < 1 || s.DailyLimit > 10000 {
		return FieldError{Field: "daily_limit", Message: "must be between 1 and 10000"}
	}
	if s.CutoffHour < 0 || s.CutoffHour > 23 {
		return FieldError{Field: "cutoff_hour", Message: "must be between 0 and 23"}
	}
	if s.Status != PortFacilityActive && s.Status != PortFacilitySuspended {
		return FieldError{Field: "status", Message: "is invalid"}
	}
	return nil
}

func (s PortFacility) BusinessDay(at time.Time) (string, error) {
	day, _, _, err := s.OperatingDayWindow(at)
	return day, err
}

func (s PortFacility) OperatingDayWindow(at time.Time) (string, time.Time, time.Time, error) {
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	local := at.In(loc)
	if local.Hour() < s.CutoffHour {
		local = local.AddDate(0, 0, -1)
	}
	day := local.Format("2006-01-02")
	start, err := time.ParseInLocation("2006-01-02 15", day+" "+fmt.Sprintf("%02d", s.CutoffHour), loc)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	return day, start.UTC(), start.AddDate(0, 0, 1).UTC(), nil
}

func (s PortFacility) IsOpen() bool { return s.Status == PortFacilityActive }

func (s PortFacility) IsSuspended() bool { return s.Status == PortFacilitySuspended }
