package domain

type Action string

const (
	ActionFleetCatalogWrite      Action = "catalog_write"
	ActionPlanVoyage             Action = "plan_voyage"
	ActionManageVoyage           Action = "manage_voyage"
	ActionResolveCatchLanding    Action = "resolve_catch_landing"
	ActionSubmitCatchDeclaration Action = "submit_catch_declaration"
	ActionReviewCatchAnomaly     Action = "review_catch_anomaly"
	ActionReadOperations         Action = "read_operations"
	ActionReadAudit              Action = "read_audit"
)

var roleActions = map[Role]map[Action]bool{
	RoleVoyageCoordinator: {ActionFleetCatalogWrite: true, ActionPlanVoyage: true, ActionManageVoyage: true, ActionResolveCatchLanding: true, ActionSubmitCatchDeclaration: true, ActionReadOperations: true, ActionReadAudit: true},
	RoleVesselCaptain:     {ActionManageVoyage: true, ActionResolveCatchLanding: true, ActionSubmitCatchDeclaration: true, ActionReadOperations: true},
	RoleFisheriesOfficer:  {ActionReviewCatchAnomaly: true, ActionReadOperations: true},
	RoleComplianceAuditor: {ActionReadOperations: true, ActionReadAudit: true},
}

func (p Principal) CanAction(action Action) bool {
	return roleActions[p.Role][action]
}
