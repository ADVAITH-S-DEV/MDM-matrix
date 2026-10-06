# MDM Matrix — Simulated Device Management

A full-stack device-management prototype built with Go, React and PostgreSQL. An administrator can monitor simulated agents, send commands, and follow acknowledgements and command history through a deployed dashboard.

**Scope:** this is a simulator-backed portfolio project, not a production MDM product. Lock, unlock, wipe and policy update commands are demonstrations; they do not control a real operating system, erase files, or apply real policies.

## Features

- Admin login using bcrypt password verification and a signed, expiring JWT.
- Device inventory with online/offline status, simulated battery level and last-seen time.
- Lock and Unlock actions with a separate lock-state display.
- Database-persisted commands for offline devices, replayed when an agent reconnects.
- Live device updates and command acknowledgements over WebSockets, supplemented by periodic HTTP refreshes.
- Recent command history and metrics: active/completed/failed counts, completion rate and average completion time.
- Concurrent simulated agents with reconnect backoff and jitter.
- Go handler/middleware tests, mocked Cypress browser tests, and GitHub Actions checks.

## Architecture

```text
React dashboard (Vercel)
   | REST: login, fleet, commands, metrics
   | WebSocket: live admin updates
   v
Go API + in-memory connection hubs (Render)
   |                         |
   | SQL via pgxpool         | Device WebSockets
   v                         v
PostgreSQL (Supabase)     Go simulator agents
```

The backend, frontend and simulator run independently. Supabase hosts PostgreSQL; login is implemented in the Go backend against the `admin_user` table, not through Supabase Auth.

## Command lifecycle

1. The dashboard sends an authenticated command request.
2. The backend saves a pending command and a creation event.
3. If the agent is connected, the backend sends the command and records delivery. Otherwise the command stays pending and a queued event is recorded.
4. The simulator waits approximately two seconds to represent execution, then acknowledges the command.
5. The backend records completion and broadcasts it to the dashboard.
6. Lock/Unlock state is derived from the latest completed lock-related command, so it survives a dashboard refresh.

An HTTP `202` means the command was accepted, not that it finished. A locked device can still be online. The simulator reports a fixed battery value rather than real telemetry.

## Repository layout

```text
backend/
  api/                 REST/WebSocket handlers, middleware and tests
  hub/                 In-memory connection registries
  types/               Request/response models
  main.go              Configuration and route setup
frontend/
  src/components/      Login, inventory, summaries and history UI
  src/hooks/           Fleet updates and command insights
  src/services/        HTTP API client
  cypress/e2e/         Mocked browser workflow tests
simulator/
  simulator.go         Concurrent demo agents
supabase/
  migrations/          Initial schema SQL
  enable_unlock.sql    Separate SQL Editor patch allowing Unlock
  seed.sql             Development-only admin seed
.github/workflows/
  ci.yml               Backend and frontend checks
```

## Local setup

Prerequisites: Go compatible with the version declared in each `go.mod`, Node.js 24 (matching CI), npm, and a configured PostgreSQL database. Use three terminals, starting from the repository root.

### Database and admin account

For a new demo database, review and apply `supabase/migrations/20261006000000_initial_schema.sql`, then `supabase/enable_unlock.sql` through the Supabase SQL Editor. For an existing database, compare its schema first; `CREATE TABLE IF NOT EXISTS` does not reconcile every existing column or constraint.

The Unlock patch is outside the migration directory: applying the initial migration alone does not enable Unlock. If SQL was already applied manually, do not blindly run `supabase db push`; reconcile the migration history before adopting CLI-managed migrations.

Admin credentials belong in `admin_user`; `password_hash` must contain a bcrypt hash. The optional `seed.sql` creates a known development account: use it only in an isolated development database, never in a hosted production/demo database exposed to others. Never publish credentials.

### Backend

Create an untracked `backend/.env` with your own values:

```dotenv
DATABASE_URL=your-postgresql-connection-string
JWT_SECRET=your-long-random-signing-secret
PORT=8080
```

```powershell
cd backend
go mod download
go run .
```

The backend defaults to port 8080. Use the exact connection string from your Supabase project, with the appropriate TLS settings. Session pooling on port 5432 is an option for IPv4 connections; Render does not inherently require port 6543. The backend uses pgx execution mode without its statement cache to support transaction pooling as well. See [Supabase connection guidance](https://supabase.com/docs/guides/database/connecting-to-postgres).

### Frontend

In another terminal:

```powershell
cd frontend
npm ci
$env:VITE_API_URL = "http://localhost:8080"
npm run dev
```

Open the URL Vite prints and sign in with your configured admin account. See the frontend README for UI configuration and testing details.

### Simulator

In a third terminal, explicitly select the local backend:

```powershell
cd simulator
$env:MDM_API_URL = "http://localhost:8080"
go mod download
go run .
```

Without `MDM_API_URL`, the simulator targets `https://mdm-matrix-backend.onrender.com`. It runs five agents named `fleet-device-001` through `fleet-device-005`.

To run one agent instead, set `DEVICE_ID` and optionally `DEVICE_NAME` before starting. Use the existing device ID to bring that dashboard row online. A row's name does not mean a physical device exists. The simulator must remain running for heartbeats and acknowledgements.

## Deployment configuration

- **Render backend:** configure `DATABASE_URL` and `JWT_SECRET` in the service environment. Render supplies `PORT`. Local changes to an `.env` file do not update hosted environment variables.
- **Vercel frontend:** configure `VITE_API_URL` as the HTTPS backend base URL, without a route suffix. It is public build-time configuration; never place database credentials or signing secrets in frontend variables.
- **Simulator:** run separately with `MDM_API_URL` pointing at the hosted backend. Hosting the API does not automatically start the simulator.

Apply required database changes before deploying dependent backend code, then deploy the frontend. Keep secrets outside Git. Changes to Vite environment values require a new frontend build.

## API overview

| Endpoint | Purpose | Authentication |
| --- | --- | --- |
| `GET /health` | Basic server liveness | None |
| `POST /login` | Verify admin credentials and return JWT | Credentials |
| `POST /enroll` | Register/re-enroll a demo agent | Currently public; prototype limitation |
| `GET /devices` | Inventory including derived lock state | Bearer JWT |
| `POST /devices/{id}/command` | Queue `lock`, `unlock`, `wipe`, or `update_policy` | Bearer JWT |
| `GET /commands?limit=20` | Recent commands (maximum limit 200) | Bearer JWT |
| `GET /metrics/commands` | Aggregate command metrics | Bearer JWT |
| `/ws?token=...` | Agent WebSocket | Device token |
| `/admin/ws?token=...` | Admin WebSocket | JWT |

Lifecycle records are stored in `command_events`. They provide a foundation for process analysis; this project does not implement a process-mining engine.

## Tests and CI

```powershell
cd backend
go vet ./...
go test ./...
go test -race ./...
```

The race detector requires a supported compiler/toolchain; CI runs it on Linux.

```powershell
cd frontend
npm ci
npm run lint
npm run build
npm run test:e2e
```

GitHub Actions runs Go formatting checks, vetting and race-enabled tests, plus frontend lint, build and Cypress. Go tests use database fakes. Cypress intercepts HTTP requests to test login, inventory, command dispatch and Unlock UI; it does not validate the real database, real agent acknowledgements or deployed WebSockets. Passing CI is not a load-test or production-readiness guarantee.

## Manual demo checklist

1. Start the API, dashboard and simulator; verify online agents.
2. Lock an agent; wait for acknowledgement and verify Locked/Unlock.
3. Unlock it; verify Unlocked/Lock.
4. Reload the dashboard; confirm the acknowledged lock state persists.
5. Stop the simulator; wait for offline detection.
6. Queue a command, restart the simulator, and verify eventual completion.
7. Check command history and aggregate metrics.

## Current limitations and next steps

- Only simulated devices; no OS-level management, policy payloads or real wipe.
- One backend instance owns its connections in memory; horizontal scaling needs shared routing/pub-sub.
- Duplicate command IDs are cached only within a simulator connection, not durably across restarts. This is not an exactly-once execution guarantee.
- Lifecycle writes are not fully transactional; duplicate acknowledgements can add duplicate event records.
- Concurrent socket writes and reconnect ordering need further hardening.
- The schema includes failed status/events, but a complete failure-reporting, timeout and retry workflow is not implemented.
- Public demo enrollment, permissive CORS/origin handling, browser token storage and tokens in WebSocket URLs require a security review before production use.
- No real-database integration suite, deployed browser suite or published performance benchmark.
- Docker packaging is a possible future enhancement; it is not currently implemented.

This project demonstrates API-driven UI development, Go concurrency, persistent command tracking, browser automation and cloud deployment. Describe it as a **simulated device-management prototype**, with these boundaries stated clearly.
