# Distant-Water Fishing Operations

Distant-Water Fishing Operations is a production-oriented Go backend for planning distant-water fishing voyages, coordinating departure and landing ports, reserving licensed vessels and support fleets, reconciling catch declarations, and completing post-landing fisheries compliance review. It uses a durable SQLite workflow and does not depend on online services.

## Architecture

- `cmd/server` wires configuration, SQLite, services, HTTP middleware, the outbox worker, and graceful shutdown.
- `cmd/seed-user` creates or updates a local operator account with a bcrypt password hash.
- `internal/domain` owns permit, vessel, support-fleet, voyage, handoff, and anomaly state machines.
- `internal/service` coordinates catalog administration, voyage planning, catch declaration, catch landing, anomaly review, authentication, and queries.
- `internal/repository` defines persistence contracts; HTTP handlers do not issue SQL.
- `internal/storage/sqlite` provides versioned migrations, transactions, optimistic updates, pagination, restart recovery, and durable job claiming.
- `internal/httpapi` exposes authenticated JSON APIs with stable error codes and request IDs.
- `internal/worker` retries outbox deliveries, expires pending handoff reviews, records permanent failures, and stops on context cancellation.
- `internal/audit` records the actor, request, action, business object, outcome, and metadata inside the owning transaction.

## Business Model

The SQLite schema contains fourteen related tables:

- `users` own revocable, expiring `sessions`. Roles are `voyage_coordinator`, `vessel_captain`, `fisheries_officer`, and `compliance_auditor`.
- `fishing_permits` define the permitted catch-variance envelope, voyage duration limit, anomaly review deadline, and business time zone.
- `port_facilities` model departure and landing ports with local-day capacity and cutoff rules.
- `fishing_vessels` track a licensed vessel by permit, departure port, registry number, vessel class, voyage count, certificate expiry, and lifecycle state.
- `support_fleets` provide certified cargo and observer capacity and can be assigned to only one active voyage.
- `fishing_voyages` bind a permit, departure port, landing port, support fleet, and one or more vessels through `fishing_voyage_vessels`.
- `catch_landing_tasks` implement expiring two-person review after landing. The voyage coordinator and assigned fisheries officer must be different people.
- `catch_declarations` carry ordered catch-variance measurements. Out-of-envelope declarations open or extend a `catch_anomaly` and quarantine affected vessels.
- `anomaly_dispositions`, `audit_events`, `idempotency_records`, and `outbox_jobs` preserve safety decisions, accountability, replay protection, and asynchronous delivery.

Voyage planning is atomic. The service checks the active permit and ports, applies the departure port's local business-day limit, validates vessel certificates and ownership, reserves every vessel, assigns a certified support fleet, persists the voyage, stores the idempotent response, enqueues an outbox job, and writes an audit event. Any intermediate failure rolls back the entire operation. Version predicates and unique constraints protect reservations from concurrent writers.

## State Machines

- Fishing permit: `draft -> active -> archived`.
- Fishing vessel: `berthed -> sea_ready -> assigned -> at_sea -> landed`, followed by `reinspected`, `quarantined`, or `retired` as allowed by the safety workflow.
- Fishing voyage: `planned -> cleared -> at_sea -> landed -> closed`; planned or cleared voyages may be cancelled.
- Support fleet: `standby -> assigned -> deployed`, plus maintenance and retirement paths.
- Catch landing: `pending -> approved|denied|expired`.
- Catch anomaly: `open -> reviewing -> cleared|rejected`.

Departing a voyage moves its assigned vessels to sea and deploys the fleet in one transaction. Cancellation releases vessel and fleet reservations. Voyage closure is rejected while a catch-landing review or catch anomaly remains unresolved.

## Configuration

Copy `.env.example` values into the process environment. No password or token is committed. Create a local operator explicitly:

```bash
DATABASE_PATH=./data/fishingops.db \
BOOTSTRAP_EMAIL=coordinator@example.test \
BOOTSTRAP_PASSWORD='change-this-local-password' \
BOOTSTRAP_DISPLAY_NAME='Voyage Coordinator' \
BOOTSTRAP_ROLE=voyage_coordinator \
go run ./cmd/seed-user
```

Start the API with `go run ./cmd/server`. Startup applies unapplied migrations. `GET /healthz` reports process liveness and `GET /readyz` checks the database. SIGINT or SIGTERM cancels workers before the HTTP shutdown deadline.

## HTTP API

Authentication uses `POST /api/v1/auth/login` and authenticated `POST /api/v1/auth/logout`. Login returns a bearer token backed by a server-side session; logout revokes it and expired sessions are rejected.

The main operational workflow includes:

- fishing permit creation and activation;
- port-facility, support-fleet, and fishing-vessel registration;
- bulk fishing-vessel registration and sea-readiness verification;
- idempotent fishing-voyage planning;
- clearance, departure, landing confirmation, closure, and cancellation;
- catch-landing creation and resolution;
- catch declaration ingestion and catch-anomaly review;
- paginated voyage, vessel, anomaly, audit, readiness, and summary queries.

Errors use one stable response shape:

```json
{"error":{"code":"business_conflict","message":"...","request_id":"req_..."}}
```

## Build And Test

```bash
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
```

Tests use temporary on-disk SQLite databases and cover migrations, rollback, optimistic conflicts, concurrent reservations, restart recovery, pagination, authentication and revocation, HTTP error contracts, lifecycle transitions, idempotency, worker retry and cancellation, and time boundaries.

Build and run the Linux image:

```bash
docker build --platform linux/amd64 -t distant-water-fishing-ops .
docker run --rm -p 8080:8080 -v fishingops-data:/data distant-water-fishing-ops
```

The image builds the real `cmd/server` entry point and stores its database at `/data/fishingops.db` by default.
