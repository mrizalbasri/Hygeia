# Hygeia Completion & Hardening Spec

Date: 2026-10-22  
Scope: Docker Backend, Session Cleanup, Patient Registration, Lab Result Correction, PDF Export, Audit Key Rotation  
Target: UINIC 8.0 Web Development Competition

---

## 1. Docker Containerization (Backend + Postgres Reference)
- Multi-stage Dockerfile (`golang:1.26-alpine` builder + `alpine:3.20` runner), non-root runtime and `.dockerignore` excluding local secrets and build archives.
- Self-contained entrypoint: auto-runs `migrate up` then `seed` (if `APP_ENV=development`) then starts API.
- `docker-compose.yml` connects to host's existing PostgreSQL container via `host.docker.internal:5434`.
- Requires `DATABASE_URL_DOCKER` from local environment; binds to `127.0.0.1:8080` and creates no database container.

## 2. Session Cleanup Scheduler
- Background goroutine in `cmd/api/main.go` runs every 1 hour.
- Executes `DELETE FROM sessions WHERE expires_at < now()`.
- Logs deleted session count; uses a 30-second query timeout and cancels on shutdown / SIGTERM.

## 3. Patient Self-Registration
- Endpoint: `POST /api/v1/auth/register` (rate-limited via standard auth bucket).
- Payload: `email`, `password` (12–1024 bytes), `full_name`, `birth_year` (1900 through the current year), `sex` (`female|male|other|unknown`). Email syntax is validated and normalized.
- Auto-generates unique `patient_ref` (`HYG-XXXX-XXXX`).
- Atomic transaction: inserts into `users` + `patients` + `sessions`, establishes authenticated session cookie immediately. Session failure rolls back the account/profile.
- Web registration is available at `/register` with an entry link from login, password confirmation, server error feedback, pending submission guard, and redirect into the authenticated patient dashboard.

## 4. Lab Result Correction (Append-Only Amendment)
- Preserves database trigger immutability on `lab_results`.
- Added migration `000010_lab_corrections.up.sql` creating table `lab_result_corrections`.
- Endpoint: `POST /api/v1/lab-results/{id}/corrections` (restricted to assigned lab organization).
- Payload: `reason` (5–500 chars), `value`, `unit`, `reference_range`, `flag`, `note`.
- Appends `lab_result.correct` event to patient's HMAC audit chain.
- `GET /api/v1/lab-results/{id}` returns chronological array of corrections.
- Corrections have UPDATE/DELETE/TRUNCATE protection. Read/query failures abort the response; emergency reads omit notes and correction reasons. Detail UI displays the original and correction history.
- Staff correction form is available in the laboratory's `Koreksi hasil` menu. A receipt link pre-fills the result UUID; manual UUID entry supports previously saved receipts without broadening result-read permissions. The form sends the complete replacement payload plus reason, preserves drafts after errors, and clears clinical fields after successful submission.

## 5. PDF Export (Lab Results & Prescriptions)
- Built directly into `ClinicalDetail.svelte` dialog using native browser `window.print()`.
- Dedicated `@media print` CSS in `workflow.css` isolates modal content, sets clean paper styles, and hides navigation / modal action controls (`.no-print`).
- Zero new npm dependencies; cross-platform PDF export via standard browser print dialog.

## 6. Audit Key Rotation
- Supports active key + historical fallback keys in `audit.Writer`.
- Configurable via `AUDIT_HMAC_KEY_PREVIOUS` environment variable.
- Supports comma-separated historical keys for multiple rotations. Invalid keys fail startup instead of being silently ignored.
- Verification algorithm: verifies digest against active key, falls back to historical keys before flagging tampering.
- Backward-compatible: prevents invalidating historical chains when rotating secrets.

---

## Verification & Status — 2026-10-24

- `go test ./cmd/... ./db/... ./internal/... -count=1`: PASS with `TEST_DATABASE_URL` against local PostgreSQL. Tests create and remove isolated schemas; demo tables are untouched.
- `go vet ./cmd/... ./db/... ./internal/...`: PASS.
- Added integration coverage: registration/session atomicity and rollback, duplicate email, expired-session cleanup preserving active sessions, lab correction authorization and immutability, and mixed-key audit chain verification.
- `pnpm check`: 0 errors, 0 warnings. `pnpm build`: PASS.
- `node tests/clinical-workflows.e2e.js`: PASS, including correction rendering, print CSS, and disabled printing after denied access.
- `node tests/provider-workflows.e2e.js`: PASS for lab, pharmacy, and admin.
- `docker build -t hygeia-api:completion .`: PASS.
- Container startup smoke test: PASS using a temporary PostgreSQL schema; migrations reached version 10 (`dirty=false`), development/test seed completed, `/readyz` returned healthy, runtime UID was non-root. Temporary container/schema were removed.
- Server deployment and production frontend adapter configuration remain unverified.

### Web form continuation — 2026-10-24

The user requested both remaining web forms. The implementation extends the
existing login and staff workspace patterns using the documented blue/cyan brand,
Plus Jakarta Sans, existing focus styles, and native form controls. No new visual
system or dependencies are introduced. The forms support desktop and narrow mobile
screens, explicit labels, API error announcements, and disabled pending controls.

- `node tests/registration.e2e.js`: PASS for validation, 409/429/503 responses,
  network failure/retry, CSRF, duplicate-submission prevention, authenticated
  redirect/reload, and widths 768/390/320 px.
- `node tests/provider-workflows.e2e.js`: PASS for receipt-to-correction navigation,
  prefilled result UUID, payload/CSRF, 403/409/503 preservation/retry, success
  reset, and mobile layout, alongside the existing provider/admin coverage.
- Desktop/mobile screenshots were inspected under `frontend/.test-results/`.
  These fixture tests verify browser behavior; backend authorization and
  atomicity are covered by the PostgreSQL integration tests above.
