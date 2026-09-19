# Pollster — Live Polling Tool

Create a poll, share the link, watch votes roll in live. No refresh needed.

---

## Project Structure

```
HCL_project/
├── backend/                  # Go (Gin) service
│   ├── cmd/server/main.go    # Entry point, wires everything together
│   ├── internal/
│   │   ├── config/           # Env-var loading
│   │   ├── models/           # Domain types (User, Poll, PollResult, …)
│   │   ├── repository/       # MongoDB + Redis data access
│   │   ├── services/         # Business logic (auth, poll, voting)
│   │   ├── handlers/         # HTTP handlers (auth, poll, SSE stream)
│   │   ├── middleware/        # JWT auth middleware
│   │   └── sse/              # SSE broker (Redis Pub/Sub fan-out)
│   ├── go.mod
│   └── .env
│
├── frontend/                 # React (Vite + TypeScript + Tailwind)
│   ├── src/
│   │   ├── api/              # Axios client + typed API calls
│   │   ├── components/ui/    # Reusable UI components
│   │   ├── context/          # AuthContext (JWT in localStorage)
│   │   ├── hooks/            # useLiveResults (SSE hook)
│   │   ├── pages/            # Home, Login, Register, CreatePoll, Poll, Dashboard
│   │   └── utils/            # fingerprint.ts, format.ts
│   ├── package.json
│   └── vite.config.ts        # Dev proxy: /api → localhost:8080
│
└── README.md
```

---

## Tech Stack

| Layer    | Technology                           | What it does                                                      |
|----------|--------------------------------------|-------------------------------------------------------------------|
| Frontend | React 18, Vite, TypeScript, Tailwind | SPA — auth, poll creation, voting, live results                   |
| Backend  | Go 1.22, Gin                         | REST API + SSE stream endpoint, input validation, JWT auth        |
| Database | MongoDB 7                            | Persists users, polls, vote counts, voter records (dedup)         |
| Realtime | Redis 7 (counters + Pub/Sub)         | Atomic vote counters; Pub/Sub drives SSE fan-out across replicas  |

---

## Prerequisites

Make sure these are installed and running before starting:

| Tool | Version | Check |
|------|---------|-------|
| Go | 1.22+ | `go version` |
| Node | 20+ | `node --version` |
| MongoDB | 7.0 | running as a Windows service |
| Redis | 5.0+ | running as a Windows service |

MongoDB and Redis run as Windows services and start automatically on boot.
To verify both are running:

```powershell
Get-Service -Name "MongoDB"
Get-Service -Name "Redis"
```

Both should show `Status: Running`.

---

## How to Run

You need **two PowerShell windows open at the same time**.

### Window 1 — Backend

```powershell
cd "C:\Users\keerthana\Desktop\HCL_project\backend"
go run ./cmd/server
```

Wait for:
```
server listening on :8080
```

### Window 2 — Frontend

```powershell
cd "C:\Users\keerthana\Desktop\HCL_project\frontend"
npm install        # first time only
npm run dev
```

Wait for:
```
VITE v5.x  ready in xxx ms
➜  Local:   http://localhost:5173/
```

Then open **http://localhost:5173** in your browser.

---

## Test the Full Flow

1. **Sign up** — create an account
2. **Create a poll** — add a question and at least 2 options
3. **Copy the poll URL** from the share box at the bottom
4. **Open that URL in a second browser tab** (or incognito)
5. **Vote in one tab** — result bars in the other tab update live, no refresh needed

---

## Environment Variables (backend/.env)

| Variable         | Default                   | Description                          |
|------------------|---------------------------|--------------------------------------|
| PORT             | 8080                      | HTTP listen port                     |
| MONGODB_URI      | mongodb://localhost:27017  | MongoDB connection string            |
| MONGODB_DB       | pollster                  | Database name                        |
| REDIS_ADDR       | localhost:6379            | Redis address                        |
| REDIS_PASSWORD   | _(empty)_                 | Redis password (if AUTH enabled)     |
| JWT_SECRET       | **required**              | HMAC-SHA256 signing key (≥ 32 chars) |
| JWT_EXPIRY_HOURS | 72                        | Token lifetime in hours              |
| ALLOWED_ORIGINS  | http://localhost:5173     | Comma-separated CORS origins         |

---

## API Reference

All endpoints are prefixed `/api/v1`.

### Auth

| Method | Path           | Auth | Description         |
|--------|----------------|------|---------------------|
| POST   | /auth/register | —    | Create account      |
| POST   | /auth/login    | —    | Login, returns JWT  |
| GET    | /auth/me       | JWT  | Current user info   |

### Polls

| Method | Path                   | Auth | Description                          |
|--------|------------------------|------|--------------------------------------|
| GET    | /polls                 | —    | List 20 most recent polls            |
| POST   | /polls                 | JWT  | Create a poll                        |
| GET    | /polls/mine            | JWT  | Polls created by the current user    |
| GET    | /polls/:id             | —    | Get poll details (live Redis counts) |
| GET    | /polls/:id/results     | —    | Get current vote results             |
| GET    | /polls/:id/stream      | —    | **SSE** — live result stream         |
| POST   | /polls/:id/vote        | —    | Cast a vote                          |
| GET    | /polls/:id/vote-status | —    | Check if this browser already voted  |
| POST   | /polls/:id/close       | JWT  | Close poll (creator only)            |

Voting requires the `X-Voter-Fingerprint` header (set automatically by the frontend).

---

## Key Design Decisions

### 1. Redis is the source of truth for live counts

Every vote does two writes:
1. **Redis `INCR`** on `poll:<id>:votes:<optionId>` and `poll:<id>:total` — instant, atomic, O(1).
2. **MongoDB `$inc`** on the poll document — durable, survives restarts.

When displaying results, the backend always merges Redis counters into the response,
falling back to MongoDB values if Redis is unavailable.

### 2. Real-time via SSE + Redis Pub/Sub

After a vote lands:
- The vote service publishes a `PollResult` JSON payload to the Redis channel `poll:<id>:results`.
- The SSE broker subscribes to that channel and fans the message out to every connected `EventSource` client.
- The browser receives the event and updates the result bars in-place — no page refresh, no polling interval.

SSE was chosen over WebSockets because results are server → client only, it works over plain
HTTP/1.1, reconnects automatically, and needs zero client libraries.

### 3. Duplicate vote prevention

Two-layer deduplication:
1. A **MongoDB unique compound index** on `{poll_id, fingerprint}` in the `voters` collection — the hard guarantee.
2. A **browser fingerprint** (random 32-char hex in `localStorage`) sent as `X-Voter-Fingerprint` on every vote request.

Voters don't need an account to vote, keeping the sharing flow frictionless.

### 4. Authentication scope

JWT + bcrypt is required only to **create** or **close** polls. Viewing and voting are
intentionally public so anyone with a shared link can participate immediately.

### 5. Backend input validation

Gin's `binding` tags enforce required fields, min/max lengths, and email format at the handler
layer. The service layer then applies business rules (duplicate options, valid option IDs,
poll-open check). The frontend also validates, but the backend never trusts client input.

### 6. Poll expiry

Polls can have an optional `ends_at` timestamp. On every `GetPoll` call the backend
checks whether the time has passed and auto-closes expired polls.
