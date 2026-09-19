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
│   ├── Dockerfile
│   ├── go.mod
│   └── .env.example
│
├── frontend/                 # React (Vite + TypeScript + Tailwind)
│   ├── src/
│   │   ├── api/              # Axios client + typed API calls
│   │   ├── components/ui/    # Reusable UI components
│   │   ├── context/          # AuthContext (JWT in localStorage)
│   │   ├── hooks/            # useLiveResults (SSE hook)
│   │   ├── pages/            # Home, Login, Register, CreatePoll, Poll, Dashboard
│   │   └── utils/            # fingerprint.ts, format.ts
│   ├── Dockerfile
│   ├── nginx.conf            # Nginx reverse-proxy config (prod)
│   └── vite.config.ts        # Dev proxy: /api → localhost:8080
│
├── docker-compose.yml        # Full stack: mongo + redis + backend + frontend
└── README.md
```

---

## Tech Stack

| Layer    | Technology                          | What it does                                                     |
|----------|-------------------------------------|------------------------------------------------------------------|
| Frontend | React 18, Vite, TypeScript, Tailwind | SPA — auth, poll creation, voting, live results                 |
| Backend  | Go 1.22, Gin                        | REST API + SSE stream endpoint, input validation, JWT auth       |
| Database | MongoDB 7                           | Persists users, polls, vote counts, voter records (dedup)        |
| Realtime | Redis 7 (counters + Pub/Sub)        | Atomic vote counters; Pub/Sub drives SSE fan-out across replicas |

---

## How to Run

### Option A — Docker Compose (recommended, one command)

**Prerequisites:** Docker Desktop running.

```bash
# 1. Set your JWT secret
#    Edit backend/.env and change JWT_SECRET to something random (≥ 32 chars)

# 2. Start everything
docker compose up --build

# Frontend → http://localhost:80
# Backend  → http://localhost:8080
# API docs  → http://localhost:8080/api/v1/...
```

All four services start with health-check gating — the backend waits for
Mongo and Redis to be ready; the frontend waits for the backend.

To stop and remove volumes:
```bash
docker compose down -v
```

---

### Option B — Local development (hot-reload)

**Prerequisites:** Go 1.22+, Node 20+, MongoDB running on 27017, Redis on 6379.

#### Backend

```bash
cd backend

# First time only — download dependencies
go mod tidy

# Copy and edit env
cp .env.example .env
# Set JWT_SECRET to something long and random

# Run
go run ./cmd/server
# Server starts on :8080
```

#### Frontend

```bash
cd frontend

npm install
npm run dev
# Vite dev server starts on http://localhost:5173
# /api requests are proxied to localhost:8080 automatically
```

Open http://localhost:5173 in your browser.

---

## API Reference

All endpoints are prefixed `/api/v1`.

### Auth

| Method | Path              | Auth | Description            |
|--------|-------------------|------|------------------------|
| POST   | /auth/register    | —    | Create account         |
| POST   | /auth/login       | —    | Login, returns JWT     |
| GET    | /auth/me          | JWT  | Current user info      |

### Polls

| Method | Path                    | Auth | Description                          |
|--------|-------------------------|------|--------------------------------------|
| GET    | /polls                  | —    | List 20 most recent polls            |
| POST   | /polls                  | JWT  | Create a poll                        |
| GET    | /polls/mine             | JWT  | Polls created by the current user    |
| GET    | /polls/:id              | —    | Get poll details (live Redis counts) |
| GET    | /polls/:id/results      | —    | Get current vote results             |
| GET    | /polls/:id/stream       | —    | **SSE** — live result stream         |
| POST   | /polls/:id/vote         | —    | Cast a vote                          |
| GET    | /polls/:id/vote-status  | —    | Check if this browser already voted  |
| POST   | /polls/:id/close        | JWT  | Close poll (creator only)            |

Voting requires the `X-Voter-Fingerprint` header (set automatically by the frontend).

---

## Key Design Decisions

### 1. Redis is the source of truth for live counts

Every vote does two writes:
1. **Redis `INCR`** on `poll:<id>:votes:<optionId>` and `poll:<id>:total` — instant, atomic, O(1).
2. **MongoDB `$inc`** on the poll document — durable, survives restarts.

When displaying results, the backend always merges Redis counters into the response,
falling back to MongoDB values if Redis is unavailable. This means results are always
as fresh as possible without an extra round-trip.

### 2. Real-time via SSE + Redis Pub/Sub (not polling, not WebSockets)

After a vote lands:
- The vote service publishes a `PollResult` JSON payload to the Redis channel `poll:<id>:results`.
- The SSE broker subscribes to that channel and fans the message out to every connected `EventSource` client.
- The browser receives the event and updates the result bars in-place — no page refresh, no polling interval.

Using Redis Pub/Sub means **multiple backend replicas stay in sync automatically** — a vote
handled by replica A is still delivered to a browser connected to replica B.

SSE was chosen over WebSockets because:
- Results are server → client only (no need for bidirectional messaging).
- SSE works over plain HTTP/1.1, reconnects automatically, and needs zero client libraries.
- Nginx proxy config is simpler (`proxy_buffering off` is all it takes).

### 3. Duplicate vote prevention

Two-layer deduplication:
1. A **MongoDB unique compound index** on `{poll_id, fingerprint}` in the `voters` collection is the hard guarantee — concurrent requests from the same browser can't both succeed.
2. The **browser fingerprint** (random 32-char hex stored in `localStorage`) is generated once per browser and sent as `X-Voter-Fingerprint` on every vote and vote-status request.

Voters don't need an account to vote, which keeps the sharing flow frictionless.

### 4. Authentication scope

Auth (JWT, bcrypt) is required only to **create** or **close** polls. Viewing and voting are
intentionally public so anyone with a shared link can participate immediately. JWTs expire
after 72 hours (configurable via `JWT_EXPIRY_HOURS`).

### 5. Input validation on the backend

Gin's `binding` tags enforce required fields, min/max lengths, and email format at the
handler layer before any data reaches the service layer. The service layer then applies
business rules (duplicate options, valid option IDs, poll-open check, multi-choice
enforcement). The frontend also validates, but the backend never trusts it.

### 6. Poll expiry

Polls can have an optional `ends_at` timestamp. On every `GetPoll` call the backend
checks whether the time has passed and auto-closes expired polls. The frontend renders
a "Closed" badge and disables the vote form accordingly.

---

## Environment Variables (backend)

| Variable          | Default                     | Description                            |
|-------------------|-----------------------------|----------------------------------------|
| PORT              | 8080                        | HTTP listen port                       |
| MONGODB_URI       | mongodb://localhost:27017   | MongoDB connection string              |
| MONGODB_DB        | pollster                    | Database name                          |
| REDIS_ADDR        | localhost:6379              | Redis address                          |
| REDIS_PASSWORD    | _(empty)_                   | Redis password (if AUTH enabled)       |
| JWT_SECRET        | **required**                | HMAC-SHA256 signing key (≥ 32 chars)   |
| JWT_EXPIRY_HOURS  | 72                          | Token lifetime in hours                |
| ALLOWED_ORIGINS   | http://localhost:5173       | Comma-separated CORS origins           |

---

## Deployment notes

The docker-compose setup is production-ready for a single-node deployment:

- The **frontend Nginx** serves the built React SPA and reverse-proxies `/api/*` to the Go
  backend, eliminating any CORS concern in production.
- The **backend** is compiled into a static binary and shipped in a `scratch` container
  (~10 MB image).
- For multi-node deployments, point all backends at a shared Redis and MongoDB instance —
  the Redis Pub/Sub broker ensures SSE delivery works across replicas without any extra
  coordination layer.

For a managed cloud deployment (e.g. Railway, Render, Fly.io):
1. Push the repo.
2. Create a MongoDB Atlas cluster and a Redis instance (Upstash or Redis Cloud free tier).
3. Set the environment variables listed above.
4. Deploy backend and frontend as separate services, update `ALLOWED_ORIGINS` to the
   frontend's public URL.
