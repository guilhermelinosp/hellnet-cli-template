# syntax=docker/dockerfile:1
# Multi-stage build: reproducible, minimal, non-root runtime.

ARG GO_VERSION=1.27

# ── Build stage ──────────────────────────────────────────────────────────────
FROM docker.io/library/golang:${GO_VERSION}-alpine AS builder

# Build metadata (overridable by CI; defaults keep local builds honest).
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /src

# Cache-friendly dependency layer.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOFLAGS=-trimpath \
    go build -ldflags="-w -s -X github.com/guilhermelinosp/golang-cli-template/internal/build.Version=${VERSION} -X github.com/guilhermelinosp/golang-cli-template/internal/build.Commit=${COMMIT} -X github.com/guilhermelinosp/golang-cli-template/internal/build.Date=${DATE}" \
    -o /app ./cmd/app

# ── Runtime stage ────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app /app

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app"]