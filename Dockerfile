# Build stage
FROM golang:1.26.5-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651 AS builder

WORKDIR /app

# Install build dependencies for CGO
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build with CGO enabled for full Kubernetes client compatibility
# Note: Using single quotes around variable values to handle special characters safely
RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w \
        -X 'github.com/pipekit/mcp-for-argo-workflows/internal/version.Version=${VERSION}' \
        -X 'github.com/pipekit/mcp-for-argo-workflows/internal/version.Commit=${COMMIT}' \
        -X 'github.com/pipekit/mcp-for-argo-workflows/internal/version.BuildTime=${BUILD_TIME}'" \
    -o mcp-for-argo-workflows \
    ./cmd/mcp-for-argo-workflows

# Runtime stage - use distroless for minimal attack surface
# Pinned by digest for reproducible builds and supply-chain stability
FROM gcr.io/distroless/base-debian12:nonroot@sha256:63f52bd27b6aa6555f5d56500b70d7bb0afe51c654905be88a2c1cf967a77b1a

# Labels for container metadata
LABEL org.opencontainers.image.title="MCP for Argo Workflows"
LABEL org.opencontainers.image.description="MCP server for Argo Workflows providing AI tool access to workflow operations"
LABEL org.opencontainers.image.source="https://github.com/pipekit/mcp-for-argo-workflows"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Copy binary from builder
COPY --from=builder /app/mcp-for-argo-workflows /mcp-for-argo-workflows

# Run as non-root user (distroless:nonroot already sets this)
USER nonroot:nonroot

# Default entrypoint
ENTRYPOINT ["/mcp-for-argo-workflows"]
