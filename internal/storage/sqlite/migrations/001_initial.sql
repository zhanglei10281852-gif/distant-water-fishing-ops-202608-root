PRAGMA foreign_keys = ON;

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_sessions_user_expiry ON sessions(user_id, expires_at);

CREATE TABLE fishing_permits (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    minimum_catch_variance_grams INTEGER NOT NULL,
    maximum_catch_variance_grams INTEGER NOT NULL,
    max_voyage_duration_seconds INTEGER NOT NULL,
    compliance_review_deadline_seconds INTEGER NOT NULL,
    business_timezone TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE port_facilities (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    status TEXT NOT NULL,
    daily_limit INTEGER NOT NULL,
    cutoff_hour INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE fishing_vessels (
    id TEXT PRIMARY KEY,
    fishing_permit_id TEXT NOT NULL REFERENCES fishing_permits(id),
    departure_port_id TEXT NOT NULL REFERENCES port_facilities(id),
    registry_number TEXT NOT NULL,
    vessel_class TEXT NOT NULL,
    voyage_count INTEGER NOT NULL,
    hold_capacity_kg INTEGER NOT NULL,
    state TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    request_id TEXT,
    quarantine_note TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(fishing_permit_id, registry_number)
);
CREATE INDEX idx_fishing_vessels_state ON fishing_vessels(state, expires_at);
CREATE INDEX idx_fishing_vessels_port_facility ON fishing_vessels(departure_port_id, created_at);
CREATE TABLE support_fleets (
    id TEXT PRIMARY KEY,
    fleet_code TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL,
    cargo_capacity_kg INTEGER NOT NULL,
    certification_due_at TEXT NOT NULL,
    last_inspected_at TEXT NOT NULL,
    assigned_voyage_id TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_support_fleets_state_certification ON support_fleets(state, certification_due_at);
CREATE TABLE fishing_voyages (
    id TEXT PRIMARY KEY,
    fishing_permit_id TEXT NOT NULL REFERENCES fishing_permits(id),
    departure_port_id TEXT NOT NULL REFERENCES port_facilities(id),
    landing_port_id TEXT NOT NULL REFERENCES port_facilities(id),
    support_fleet_id TEXT NOT NULL REFERENCES support_fleets(id),
    voyage_code TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL,
    departure_window_opens_at TEXT NOT NULL,
    landing_deadline_at TEXT NOT NULL,
    departed_at TEXT,
    landed_at TEXT,
    closed_at TEXT,
    total_hold_capacity_kg INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_fishing_voyages_route_window ON fishing_voyages(departure_port_id, landing_port_id, departure_window_opens_at);
CREATE INDEX idx_fishing_voyages_state ON fishing_voyages(state, landing_deadline_at);
CREATE TABLE fishing_voyage_vessels (
    request_id TEXT NOT NULL REFERENCES fishing_voyages(id) ON DELETE CASCADE,
    fishing_vessel_id TEXT NOT NULL UNIQUE REFERENCES fishing_vessels(id),
    added_at TEXT NOT NULL,
    PRIMARY KEY(request_id, fishing_vessel_id)
);
CREATE TABLE catch_landing_tasks (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES fishing_voyages(id),
    coordinator_id TEXT NOT NULL,
    fisheries_officer_id TEXT NOT NULL,
    landing_station TEXT NOT NULL,
    status TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    resolved_at TEXT,
    resolution_note TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_catch_landing_task_pending_run ON catch_landing_tasks(request_id) WHERE status = 'pending';
CREATE INDEX idx_catch_landing_task_expiry ON catch_landing_tasks(status, expires_at);
CREATE TABLE catch_declarations (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES fishing_voyages(id),
    species_code TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    catch_variance_grams INTEGER NOT NULL,
    recorded_at TEXT NOT NULL,
    received_at TEXT NOT NULL,
    UNIQUE(request_id, species_code, sequence)
);
CREATE INDEX idx_declarations_run_time ON catch_declarations(request_id, recorded_at);
CREATE TABLE catch_anomalies (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES fishing_voyages(id),
    status TEXT NOT NULL,
    first_declaration_at TEXT NOT NULL,
    last_declaration_at TEXT NOT NULL,
    minimum_catch_variance_grams INTEGER NOT NULL,
    maximum_catch_variance_grams INTEGER NOT NULL,
    declaration_count INTEGER NOT NULL,
    review_due_at TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_catch_anomaly_active_run ON catch_anomalies(request_id) WHERE status IN ('open', 'reviewing');
CREATE INDEX idx_catch_anomaly_review_due ON catch_anomalies(status, review_due_at);
CREATE TABLE anomaly_dispositions (
    id TEXT PRIMARY KEY,
    catch_anomaly_id TEXT NOT NULL REFERENCES catch_anomalies(id),
    reviewer_id TEXT NOT NULL,
    decision TEXT NOT NULL,
    rationale TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    metadata_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_audit_entity ON audit_events(entity_type, entity_id, created_at);
CREATE INDEX idx_audit_request ON audit_events(request_id);
CREATE TABLE idempotency_records (
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_code INTEGER NOT NULL,
    response_body BLOB NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY(scope, idempotency_key)
);
CREATE TABLE outbox_jobs (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload BLOB NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL,
    available_at TEXT NOT NULL,
    locked_at TEXT,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_outbox_claim ON outbox_jobs(status, available_at, created_at);
