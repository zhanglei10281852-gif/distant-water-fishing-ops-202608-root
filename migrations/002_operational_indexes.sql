CREATE INDEX idx_fishing_vessels_run_state ON fishing_vessels(request_id, state);
CREATE INDEX idx_fishing_voyages_workspace_created ON fishing_voyages(fishing_permit_id, created_at);
CREATE INDEX idx_catch_landing_tasks_custodians ON catch_landing_tasks(fisheries_officer_id, status, created_at);
CREATE INDEX idx_jobs_aggregate ON outbox_jobs(kind, aggregate_id, created_at);
