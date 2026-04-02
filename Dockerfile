# UI build stage - Bun for fast installs, Node for webpack build
FROM node:22-alpine AS ui-builder
RUN npm i -g bun
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --ignore-scripts
COPY ui/ ui/
RUN cd ui && node ../node_modules/@vue/cli-service/bin/vue-cli-service.js build

# Go build stage with CGO enabled
FROM golang:1.24-alpine AS builder

ARG VERSION=CURRENT
ARG COMMIT=HEAD
ARG DATE=unknown

WORKDIR /app

# Install build deps (musl-dev for CGO + SQLite on Alpine)
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Cache Go modules separately
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui-builder /app/ui/dist ./ui/dist
RUN CGO_ENABLED=1 go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" -o xbvr

# Runtime stage - Alpine for minimal image size
FROM alpine:3.21

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
