# Build stage with CGO enabled
FROM golang:1.24-bookworm AS builder

# Build arguments for version info
ARG VERSION=CURRENT
ARG COMMIT=HEAD
ARG DATE=unknown

WORKDIR /app
COPY . .

# Install build dependencies
RUN apt-get update && \
    apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    pkg-config

# Build with CGO enabled and version info
RUN CGO_ENABLED=1 go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" -o xbvr

# Runtime stage
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
    libxext6 \
    libsm6 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/xbvr /usr/bin/xbvr
RUN chmod +x /usr/bin/xbvr

EXPOSE 9998-9999
VOLUME /root/.config/

CMD ["/usr/bin/xbvr"]
