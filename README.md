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

---

## ⚙️ How to Run Locally

You will need three terminal windows to run the full distributed system simultaneously.

### 1. Start the Backend
\`\`\`bash
cd backend
go mod tidy
go run main.go
\`\`\`
*(Requires a `.env` file containing your `DATABASE_URL`)*

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
*(Spins up concurrent device goroutines that automatically enroll, connect via WebSockets, and report heartbeats)*