# syntax=docker/dockerfile:1

# ── UI build stage ──────────────────────────────────────────────────
FROM node:22-alpine AS ui-builder
RUN npm i -g bun
WORKDIR /app
COPY package.json bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile --ignore-scripts
COPY ui/ ui/
RUN cd ui && bunx vite build

# ── Go build stage ──────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

ARG VERSION=CURRENT
ARG COMMIT=HEAD
ARG DATE=unknown

WORKDIR /app

RUN apk add --no-cache gcc musl-dev sqlite-dev

# Cache Go modules separately (changes rarely)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
COPY --from=ui-builder /app/ui/dist ./ui/dist

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" -o xbvr

# ── Runtime stage ───────────────────────────────────────────────────
FROM alpine:3.22

RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    ffmpeg \
    curl \
    bash

COPY --from=builder /app/xbvr /usr/bin/xbvr
COPY --from=builder /app/xbvr_data /tmp/xbvr_data
COPY docker_start.sh /docker_start.sh
RUN chmod +x /usr/bin/xbvr /docker_start.sh

EXPOSE 9998-9999
VOLUME /root/.config/

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -sf http://localhost:9999/ || exit 1

CMD ["/docker_start.sh"]
