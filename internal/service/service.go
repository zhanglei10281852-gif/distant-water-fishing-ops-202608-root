package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/audit"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/clock"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

type Services struct {
	Auth             *AuthService
	Catalog          *CatalogService
	SupportFleets    *SupportFleetService
	FishingVoyages   *VoyageService
	Handoffs         *HandoffService
	CatchDeclaration *CatchDeclarationService
	Review           *ReviewService
	Query            *QueryService
}

type dependencies struct {
	store repository.Store
	clock clock.Clock
	audit audit.Recorder
}

func New(store repository.Store, c clock.Clock, sessionTTL, catch_landing_taskTTL time.Duration) *Services {
	deps := dependencies{store: store, clock: c, audit: audit.NewRecorder(c)}
	return &Services{
		Auth:             &AuthService{dependencies: deps, sessionTTL: sessionTTL},
		Catalog:          &CatalogService{dependencies: deps},
		SupportFleets:    &SupportFleetService{dependencies: deps},
		FishingVoyages:   &VoyageService{dependencies: deps},
		Handoffs:         &HandoffService{dependencies: deps, catch_landing_taskTTL: catch_landing_taskTTL},
		CatchDeclaration: &CatchDeclarationService{dependencies: deps},
		Review:           &ReviewService{dependencies: deps},
		Query:            &QueryService{dependencies: deps},
	}
}

func requireRole(principal domain.Principal, roles ...domain.Role) error {
	if principal.UserID == "" {
		return fmt.Errorf("authentication required: %w", domain.ErrValidation)
	}
	if !principal.Can(roles...) {
		return fmt.Errorf("role %s is not permitted: %w", principal.Role, domain.ErrConflict)
	}
	return nil
}

func requireAction(principal domain.Principal, action domain.Action) error {
	if principal.UserID == "" {
		return fmt.Errorf("authentication required: %w", domain.ErrValidation)
	}
	if !principal.CanAction(action) {
		return fmt.Errorf("role %s cannot perform %s: %w", principal.Role, action, domain.ErrConflict)
	}
	return nil
}

func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }
