# ---------------------------------------------------------------------------
# Stage 1: Frontend — build the SvelteKit static site
# ---------------------------------------------------------------------------
FROM node:22-alpine AS frontend

WORKDIR /ui

# Install bun
RUN npm install -g bun

# Copy dependency manifests first (better layer caching)
COPY ui/package.json ui/bun.lock ./
COPY ui/svelte.config.js ./
COPY ui/vite.config.ts ./
COPY ui/tsconfig.json ./
COPY ui/components.json ./

# Install frontend dependencies
RUN bun install --frozen-lockfile

# Generate SvelteKit types and .svelte-kit directory (required for build)
# The "prepare" script runs svelte-kit sync which generates the needed files
RUN bun run prepare

# Copy the frontend source needed for the build
COPY ui/src/ ./src/
COPY ui/static/ ./static/

# Build — output goes to /ui/build
RUN bun run build

# ---------------------------------------------------------------------------
# Stage 2: Builder — compile the Go binary
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build \
	-a -installsuffix cgo \
	-o euro-haus ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux go build \
	-a -installsuffix cgo \
	-o create-admin ./cmd/create-admin

RUN CGO_ENABLED=0 GOOS=linux go build \
	-a -installsuffix cgo \
	-o migrate-stripe ./cmd/migrate-stripe

# ---------------------------------------------------------------------------
# Stage 3: Prod — minimal image with binary + static assets
# ---------------------------------------------------------------------------
FROM alpine:latest AS prod

WORKDIR /app

# Copy the Go binary
COPY --from=builder /app/euro-haus .
COPY --from=builder /app/create-admin .
COPY --from=builder /app/migrate-stripe .

# Copy the SvelteKit build output
COPY --from=frontend /ui/build ./static/

ENV STATIC_DIR=./static

HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:8080/healthcheck || exit 1

EXPOSE 8080
CMD ["./euro-haus"]
