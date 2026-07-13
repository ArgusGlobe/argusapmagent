# syntax=docker/dockerfile:1.7

# Run the Go compiler using the builder machine's native architecture.
# On an Apple Silicon Mac, this stage runs as linux/arm64.
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build

WORKDIR /src

# git is required when Go needs to download a module directly from Git.
RUN apk add --no-cache \
    ca-certificates \
    git

# Docker automatically provides these when using Buildx.
ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./

ENV GOPROXY=https://proxy.golang.org,direct

# Cache downloaded Go modules between builds.
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Build a static Linux binary for the platform supplied by:
# docker buildx build --platform ...
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/probe-ecs-agent \
      ./cmd/agent

# This final stage is built for TARGETPLATFORM, such as linux/amd64.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=build /etc/ssl/certs/ca-certificates.crt \
    /etc/ssl/certs/ca-certificates.crt

COPY --from=build /out/probe-ecs-agent /probe-ecs-agent

EXPOSE 8080

# Useful for local Docker health reporting.
# For ECS, also define this health check in the task definition.
HEALTHCHECK --interval=30s \
    --timeout=3s \
    --start-period=10s \
    --retries=3 \
    CMD ["/probe-ecs-agent", "healthcheck"]

USER nonroot:nonroot

ENTRYPOINT ["/probe-ecs-agent"]
