# MDM-Matrix: Real-Time Mobile Device Management Platform

A high-performance, real-time Mobile Device Management (MDM) prototype engineered to remotely monitor, manage, and push operational commands to a concurrent fleet of endpoints. Built with a heavy focus on distributed systems resilience, concurrency patterns, and algorithmic optimization.

## 🏗️ System Architecture

The platform uses a decoupled architecture optimized for low-latency state synchronization and minimal database I/O:

*   **Backend:** Go (Modularized into clean `api`, `hub`, and `types` packages)
*   **Frontend:** React (Vite) with custom hooks for WebSocket state management
*   **Database:** PostgreSQL with connection pooling (`pgxpool`)
*   **Simulated Fleet:** Go Goroutines simulating concurrent remote device agents

---

## 🚀 Key Engineering & Optimization Features

Designed from the ground up to solve classic distributed systems challenges and handle network partitions gracefully:

*   **$O(1)$ WebSocket Hub Routing:** Active connections are managed in-memory and protected by a `sync.RWMutex`. Adding, removing, and broadcasting to connected devices operates in $O(1)$ time complexity with a strictly bounded memory footprint.
*   **$O(\log N)$ Offline Command Queueing:** Commands targeted at offline devices are safely persisted in PostgreSQL. A composite B-Tree index on `(device_id, status)` optimizes command queue retrieval upon device reconnection, dropping scan complexity from $O(N)$ to $O(\log N)$.
*   **Thundering Herd Protection (Exponential Backoff with Jitter):** Device agents feature exponential backoff with randomized jitter. If the server restarts, reconnect attempts are scattered across time to prevent database connection pool exhaustion and CPU spikes.
*   **Idempotent Receivers (At-Least-Once Delivery):** The device simulator maintains a thread-safe local cache of executed command IDs. If a network blip causes the backend to re-transmit a command, the device rejects the duplicate in $O(1)$ time, guaranteeing safe, idempotent execution.
*   **Phantom Device Mitigation:** Enforces a strict `ReadDeadline` (Heartbeat Timeout) on server sockets. If a device experiences a dirty TCP disconnect, the backend safely reaps the connection and updates its state in $O(1)$ time, eliminating memory leaks.
*   **Real-Time Admin Dashboard:** Shifted the React frontend from HTTP short-polling to a live WebSocket Pub/Sub model. This eliminates redundant database queries, pushing real-time UI state changes instantly in $O(K)$ time (where $K$ is active admin sessions).

---

## 📂 Project Structure

\`\`\`bash
MDM-Matrix/
├── backend/       # Go REST & WebSocket Server (JWT auth, Hub, Handlers)
├── frontend/      # React Admin Dashboard (Vite, Custom Hooks, Component Architecture)
└── simulator/     # Go Device Fleet Agent (Goroutines, Backoff, Idempotency)
\`\`\`

Database migrations live in `supabase/migrations`, browser automation lives in
`frontend/cypress`, and GitHub Actions validates backend and frontend changes on
every push and pull request.

---

## ⚙️ How to Run Locally

You will need three terminal windows to run the full distributed system simultaneously.

### 1. Start the Backend
\`\`\`bash
cd backend
go mod tidy
go run main.go
\`\`\`
The backend requires these environment variables:

```env
DATABASE_URL=postgresql://...
JWT_SECRET=generate-a-long-random-secret
```

For Render, use the exact Supabase connection string shown under **Connect →
Transaction pooler**. It uses port `6543`, an IPv4-compatible pooler hostname,
and a username in the form `postgres.PROJECT_REF`. The backend disables pgx's
prepared-statement cache because Supabase transaction mode does not support it.

Admin credentials are read from the Supabase `admin_user` table. Its
`password_hash` value must be a bcrypt hash; plaintext passwords are never stored
in Render or in the database. Keep the same `JWT_SECRET` across deploys so
existing login tokens remain valid.

For a Render deployment, set both values in the service's Environment page.
Render supplies `PORT` automatically. For Vercel, set `VITE_API_URL` to the full
HTTPS Render service URL (with no `/login` suffix), then redeploy the frontend.

### 2. Start the Admin Dashboard
\`\`\`bash
cd frontend
npm install
npm run dev
\`\`\`
*(Access the dashboard at `http://localhost:5173`. Log in with your admin credentials)*

### 3. Start the Device Simulator
\`\`\`bash
cd simulator
go mod tidy
go run simulator.go
\`\`\`
Set `MDM_API_URL` only when targeting a different backend; it defaults to the
hosted Render service. For local development, set it to `http://localhost:8080`.
The simulator spins up concurrent device goroutines that enroll, connect through
WebSockets, execute demo commands, and report heartbeats.

To run one named device instead of the five-device fleet, set `DEVICE_ID` and
`DEVICE_NAME` before starting it. The ID must match the device row shown in the
dashboard.

## Database migrations

The repository follows Supabase's versioned migration workflow. Before deploying
the backend changes, link the Supabase CLI to the project and apply the migration:

\`\`\`bash
supabase link --project-ref YOUR_PROJECT_REF
supabase db push
\`\`\`

The migration creates the reproducible schema, queue indexes, row-level security,
and the append-only `command_events` lifecycle log. The backend connects with the
database service role; the browser has no direct table access.

## Tests and automation

\`\`\`bash
cd backend
go test ./...

cd ../frontend
npm ci
npm run lint
npm run build
npm run test:e2e
\`\`\`

The Cypress flow covers login, fleet loading, and command dispatch. GitHub Actions
runs formatting, vetting, race-enabled Go tests, frontend lint/build, and Cypress.

## Command process API

- `GET /commands?limit=20` returns recent command history.
- `GET /metrics/commands` returns active/completed/failed counts and average cycle time.
- `command_events` records created, queued, delivered, acknowledged, completed, and failed activities for process-mining analysis.
