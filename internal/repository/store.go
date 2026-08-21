package repository

import (
	"context"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
)

type Store interface {
	WithTx(ctx context.Context, fn func(Tx) error) error
	Read(ctx context.Context, fn func(Reader) error) error
	Ping(ctx context.Context) error
	Close() error
}

type Reader interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUser(ctx context.Context, id string) (domain.User, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error)
	GetFishingPermit(ctx context.Context, id string) (domain.FishingPermit, error)
	GetPortFacility(ctx context.Context, id string) (domain.PortFacility, error)
	GetFishingVessel(ctx context.Context, id string) (domain.FishingVessel, error)
	GetSupportFleet(ctx context.Context, id string) (domain.SupportFleet, error)
	GetFishingVoyage(ctx context.Context, id string) (domain.FishingVoyage, error)
	ListFishingVoyageVessels(ctx context.Context, requestID string) ([]domain.FishingVessel, error)
	GetPendingCatchLanding(ctx context.Context, requestID string) (domain.CatchLandingTask, error)
	GetCatchLandingTask(ctx context.Context, id string) (domain.CatchLandingTask, error)
	GetActiveCatchAnomaly(ctx context.Context, requestID string) (domain.CatchAnomaly, error)
	GetCatchAnomaly(ctx context.Context, id string) (domain.CatchAnomaly, error)
	GetVoyageReadiness(ctx context.Context, requestID string) (domain.VoyageReadiness, error)
	GetPlatformSummary(ctx context.Context) (PlatformSummary, error)
	ListFishingVoyages(ctx context.Context, filter FishingVoyageFilter) (FishingVoyagePage, error)
	ListFishingVessels(ctx context.Context, filter FishingVesselFilter) (FishingVesselPage, error)
	ListLandingAnomalies(ctx context.Context, filter CatchAnomalyFilter) (CatchAnomalyPage, error)
	ListAuditEvents(ctx context.Context, filter AuditFilter) (AuditPage, error)
	GetIdempotency(ctx context.Context, scope, key string) (IdempotencyRecord, error)
	CountPortFacilityFishingVoyagesForWindow(ctx context.Context, portFacilityID string, startsAt, endsAt time.Time) (int, error)
}

type Tx interface {
	Reader
	InsertUser(ctx context.Context, user domain.User) error
	InsertSession(ctx context.Context, session domain.Session) error
	RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error
	InsertFishingPermit(ctx context.Context, program domain.FishingPermit) error
	UpdateFishingPermit(ctx context.Context, program domain.FishingPermit, expectedVersion int64) error
	InsertPortFacility(ctx context.Context, port_facility domain.PortFacility) error
	InsertFishingVessel(ctx context.Context, batch domain.FishingVessel) error
	UpdateFishingVessel(ctx context.Context, batch domain.FishingVessel, expectedVersion int64) error
	InsertSupportFleet(ctx context.Context, support_fleet domain.SupportFleet) error
	UpdateSupportFleet(ctx context.Context, support_fleet domain.SupportFleet, expectedVersion int64) error
	InsertFishingVoyage(ctx context.Context, run domain.FishingVoyage) error
	UpdateFishingVoyage(ctx context.Context, run domain.FishingVoyage, expectedVersion int64) error
	InsertFishingVoyageVesselLink(ctx context.Context, item domain.FishingVoyageVesselLink) error
	InsertCatchLandingTask(ctx context.Context, catch_landing_task domain.CatchLandingTask) error
	UpdateCatchLandingTask(ctx context.Context, catch_landing_task domain.CatchLandingTask, expectedVersion int64) error
	InsertCatchDeclaration(ctx context.Context, declaration domain.CatchDeclaration) error
	InsertCatchAnomaly(ctx context.Context, catch_anomaly domain.CatchAnomaly) error
	UpdateCatchAnomaly(ctx context.Context, catch_anomaly domain.CatchAnomaly, expectedVersion int64) error
	InsertAnomalyDisposition(ctx context.Context, decision domain.AnomalyDisposition) error
	InsertAuditEvent(ctx context.Context, event domain.AuditEvent) error
	PutIdempotency(ctx context.Context, record IdempotencyRecord) error
	InsertJob(ctx context.Context, job domain.OutboxJob) error
	ClaimJobs(ctx context.Context, now time.Time, limit int) ([]domain.OutboxJob, error)
	CompleteJob(ctx context.Context, id string, now time.Time) error
	RetryJob(ctx context.Context, id string, availableAt time.Time, lastError string, dead bool) error
	ExpireCatchLandingTasks(ctx context.Context, now time.Time, limit int) ([]domain.CatchLandingTask, error)
}

type PageRequest struct {
	Limit  int
	Offset int
	Sort   string
	Desc   bool
}

func (p PageRequest) Normalize(max int) PageRequest {
	if p.Limit < 1 {
		p.Limit = 50
	}
	if p.Limit > max {
		p.Limit = max
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

type FishingVoyageFilter struct {
	Page            PageRequest
	FishingPermitID string
	DeparturePortID string
	LandingPortID   string
	State           domain.FishingVoyageState
	From            *time.Time
	To              *time.Time
}

type FishingVoyagePage struct {
	Items []domain.FishingVoyage
	Total int
}

type FishingVesselFilter struct {
	Page            PageRequest
	FishingPermitID string
	PortFacilityID  string
	FishingVoyageID string
	State           domain.FishingVesselState
	ExpiresBy       *time.Time
}

type FishingVesselPage struct {
	Items []domain.FishingVessel
	Total int
}

type CatchAnomalyFilter struct {
	Page            PageRequest
	FishingVoyageID string
	Status          domain.CatchAnomalyStatus
	DueBefore       *time.Time
}

type CatchAnomalyPage struct {
	Items []domain.CatchAnomaly
	Total int
}

type AuditFilter struct {
	Page       PageRequest
	EntityType string
	EntityID   string
	Actor      string
	RequestID  string
}

type AuditPage struct {
	Items []domain.AuditEvent
	Total int
}

type PlatformSummary struct {
	FishingPermitsActive        int `json:"fishing_permits_active"`
	FishingVesselsSeaReady      int `json:"fishing_vessels_sea_ready"`
	FishingVesselsMaterializing int `json:"fishing_vessels_at_sea"`
	FishingVesselsQuarantined   int `json:"fishing_vessels_quarantined"`
	SupportFleetsStandby        int `json:"support_fleets_standby"`
	FishingVoyagesActive        int `json:"fishing_voyages_active"`
	OpenLandingAnomalies        int `json:"open_catch_anomalies"`
	PendingCatchLandings        int `json:"pending_catch_landings"`
	FailedJobs                  int `json:"failed_jobs"`
}

type IdempotencyRecord struct {
	Scope        string
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
