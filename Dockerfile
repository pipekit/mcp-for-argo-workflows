# Build stage
FROM golang:1.27.0-bookworm@sha256:484ef6066fa69acb059fdfeda7ba2b8f7391f2ef6abc6f9b8411e669ebd56466 AS builder

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
FROM gcr.io/distroless/base-debian12:nonroot@sha256:7f0c72cd138b442ae0deeb69c08b1acf5525439ba251a49ad93c320a061567e5

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
