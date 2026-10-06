# MDM Matrix frontend

React/Vite admin dashboard for the simulator-backed device-management prototype. The Go backend provides authentication, inventory, command APIs and live WebSocket events. The browser does not connect directly to PostgreSQL.

## Run locally

Use Node.js 24 and npm, matching GitHub Actions.

```powershell
npm ci
$env:VITE_API_URL = "http://localhost:8080"
npm run dev
```

Alternatively, set `VITE_API_URL=http://localhost:8080` in an untracked `.env.local`. Restart Vite after changing it. Without this variable, the app targets `http://localhost:8080`.

For hosted builds, set `VITE_API_URL` to the HTTPS backend base URL and rebuild. Do not append `/login` or another route. Vite embeds this value in browser code: it is not a place for secrets.

## Dashboard behaviour

- Login stores the admin JWT in browser local storage; Logout removes it.
- Inventory loads over HTTP and receives live updates over the admin WebSocket.
- Periodic HTTP refreshes supplement live updates.
- Commands show sending, awaiting-device or queued feedback before completion.
- Lock changes to Unlock only after acknowledgement; connectivity is shown separately.
- Persisted completed command history restores lock state after reload.
- History includes aggregate completion statistics and a manual refresh control.

All device operations are simulated. Wipe does not erase files, and Policy update does not configure a real device.

## Source layout

```text
src/
  components/   LoginForm, DeviceTable, DashboardHeader, FleetStats, CommandHistory
  hooks/        Fleet subscription and command-insights fetching
  services/     HTTP API client
  App.jsx       Authentication and command-feedback coordination
  App.css       UI styling
cypress/e2e/    Mocked browser workflow tests
public/        Application logo
```

## Quality checks

```powershell
npm run lint
npm run build
npm run test:e2e
```

The E2E script starts Vite at `http://127.0.0.1:5173`, runs Cypress and stops the test server. Leave that port available. Cypress may download its browser-runner binary on first installation.

The tests stub HTTP responses to verify login, fleet rendering, command request payloads and Unlock availability for a locked device. They do not test the actual Go backend, Supabase database or simulator WebSocket acknowledgement flow. Use the root README's manual checklist to verify the full system.

Shared Cypress support setup is disabled because these tests do not require a support file. Legacy `Cypress.env()` access is also disabled.

## Troubleshooting

- **401:** verify the admin account/password and token expiry. A failed login and an expired dashboard token are different cases.
- **500:** inspect backend logs and database schema/configuration.
- **CORS/preflight errors:** inspect the backend OPTIONS response and configured API URL; do not disable browser security.
- **Offline devices:** start the simulator against the same backend as the dashboard.
- **Queued/pending commands:** confirm an agent is connected and can acknowledge; HTTP acceptance alone is not completion.
- **Old deployed configuration:** rebuild/redeploy after changing `VITE_API_URL`.

See the root README for architecture, database setup, simulator commands and prototype limitations.
