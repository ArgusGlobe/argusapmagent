# Probe ECS Fargate Sidecar Agent

Go sidecar agent for ECS Fargate tasks. It runs beside the application container,
discovers task/container identity from the ECS Task Metadata Endpoint v4, collects
container resource telemetry, and streams batches to Hermes over gRPC.

## Design

- No AWS SDK calls.
- No CloudWatch Metrics API.
- No manually configured agent id, task arn, cluster name, or project id.
- Stable per-task Agent ID generated from ECS runtime metadata.
- Local mock mode when `ECS_CONTAINER_METADATA_URI_V4` is unavailable.
- gRPC client-streaming with retry and gzip compression.
- Clean packages for config, metadata, collector, telemetry, grpcclient, health.

## Environment Variables

Only these variables are used:

| Variable | Purpose |
| --- | --- |
| `HERMES_HOST` | Hermes gRPC host. Defaults to `127.0.0.1`. |
| `HERMES_GRPC_PORT` | Hermes gRPC port. Defaults to `19090`. |
| `PROBE_ENVIRONMENT` | Environment label such as `prod`, `uat`, `dev`. |
| `PROBE_SERVICE_NAME` | Human service name shown for the probe. |
| `PROBE_COLLECTION_INTERVAL_SECONDS` | Collection interval. Minimum 5s, default 30s. |
| `ECS_CONTAINER_METADATA_URI_V4` | Injected automatically by ECS Fargate. |

## Local Development

Run without ECS metadata:

```bash
go run ./cmd/agent
```

The agent will switch to mock mode and emit mock container telemetry.

Run tests:

```bash
go test ./...
```

## Docker

```bash
docker build -t probe-ecs-fargate-agent:latest .
```

## ECS Deployment

Use `deploy/ecs/task-definition.example.json` as the pattern. Add the
`probe-sidecar` container to every Fargate task definition. ECS automatically
injects `ECS_CONTAINER_METADATA_URI_V4`.

## Hermes Compatibility

The agent uses the existing Hermes `argus.hermes.v1.Ingest` gRPC service
and sends:

- `agent_id`
- cluster
- service
- task ARN
- launch type
- container samples
- log lines
- sent timestamp

The new product requirement says no manual token or registration. Therefore the
sidecar does not require an auth token. Hermes should accept first-batch
self-registration for the generated Agent ID, or run this channel behind a
private network boundary/mTLS.

## Logs

ECS Fargate does not expose sibling container stdout files directly to another
container. For now the agent supports the telemetry log model and local mock
logs. Production log shipping should be added in one of these ways:

1. Application writes structured logs to a shared volume mounted into the
   sidecar.
2. Application sends OTLP logs to the sidecar.
3. FireLens routes logs to a Hermes-compatible endpoint.

This is not the CloudWatch Metrics API. FireLens is part of ECS logging
plumbing; destination cost depends on where logs are sent.

## ADOT/OpenTelemetry Direction

This agent follows OpenTelemetry-style semantic labels and keeps metrics/logs in
an extensible internal model. Next step is to add an OTLP receiver/exporter path
so application metrics/logs/traces can flow through the sidecar without changing
the Hermes gRPC shipper.
