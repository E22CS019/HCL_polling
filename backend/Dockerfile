# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Cache dependency downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

# Build a statically-linked binary so it runs on any Linux base image
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /pollster ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
# Use Alpine for a minimal but functional image (wget available for healthcheck)
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /pollster /pollster

EXPOSE 8080
ENTRYPOINT ["/pollster"]
