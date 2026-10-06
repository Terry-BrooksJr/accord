# Accord

An integration platform built with Go and Python, designed to keep independent external systems consistent through reliable webhook ingestion, failure recovery, and reconciliation.

## Project Status

Accord is in early development. The repository currently establishes the project structure and development tooling. Application services, infrastructure configuration, and integrations are being implemented incrementally.

## Purpose

External systems can send duplicate events, deliver updates out of order, become unavailable, or disagree about transaction state.

Accord is being built to accept these unreliable inputs and converge toward a consistent, observable, recoverable internal state.

The project focuses on practical integration engineering:

- External API integrations and authentication
- Webhook verification and idempotency
- Event journaling and replay
- Classified failures and scheduled retries
- Reconciliation with recorded mismatch explanations
- Incremental ETL and integration analytics
- Production troubleshooting and observability

## Architecture

Accord uses three primary application services:

| Component | Responsibility |
| --- | --- |
| Go Webhook Gateway | Receive and verify webhooks, detect duplicates, and persist events |
| Python Integration API | Expose platform APIs, provider operations, and administrative controls |
| Python Integration Worker | Process workflows, retries, outbox events, and reconciliation jobs |

PostgreSQL serves as the transactional source of truth. A transactional outbox will connect business-state changes to background processing.

ClickHouse will store integration analytics populated through an incremental ETL pipeline.

## Planned Integrations

| Provider | Purpose |
| --- | --- |
| Stripe | Payments, refunds, and payouts |
| Shippo | Shipping and tracking |
| Plaid | Banking data and transactions |
| Zendesk or a mock support provider | Support tickets |
| Keycloak | Authentication and role-based access |

Provider-specific behavior lives in adapters so application workflows can remain independent of vendor implementations.

## Repository Structure

```text
.
├── services/
│   ├── webhook-gateway/
│   ├── integration-api/
│   └── integration-worker/
├── packages/python/
│   ├── accord-core/
│   └── accord-integrations/
├── database/
│   ├── postgresql/
│   └── clickhouse/
├── etl/
├── infrastructure/
├── docs/
│   ├── adr/
│   ├── api/
│   ├── integrations/
│   └── runbooks/
├── scenarios/
├── fixtures/
├── tests/
├── scripts/
├── .devcontainer/
└── .github/workflows/
```

## Development

Development tooling includes:

- Go
- Python 3.12
- uv
- Task
- Docker with Docker Compose

A development container configuration is available in `.devcontainer/devcontainer.json`.

List available commands:

```bash
task
```

As the corresponding configuration and application files are implemented:

```bash
# Install Python project dependencies
task python:setup

# Start local infrastructure
task infra:up

# Run each service in a separate terminal
task gateway:run
task api:run
task worker:run

# Format code and run checks
task fmt
task check
```

Scaffolded files are empty placeholders. These commands require valid project manifests, Compose configuration, and application entry points.

Use sandbox accounts and local fixtures during development. Keep credentials out of version control; document required variables in `.env.example`.

## Planned Infrastructure

Local development and the initial VPS deployment will use Docker Compose with:

- Traefik
- PostgreSQL
- ClickHouse
- Keycloak
- Prometheus
- Grafana
- Loki

Logs and persisted events will carry correlation IDs to support troubleshooting across services.

## Integration Scenarios

Deterministic scenarios will demonstrate expected behavior under normal operation and failures:

- Happy path
- Duplicate webhook
- Payment or shipping failure
- Provider timeout
- Payout mismatch
- Refund after shipping
- Reconciliation success and failure

Scenario definitions, fixtures, and expected outcomes live in `scenarios/` and `fixtures/`.

## Documentation

- [Architecture](ARCHITECTURE.md)
- [Development](DEVELOPMENT.md)
- [Deployment](DEPLOYMENT.md)
- [Operations](OPERATIONS.md)
- [Security](SECURITY.md)
- [Architecture Decisions](docs/adr/)

These documents will grow alongside the implementation.

## Roadmap

1. Local infrastructure and Go webhook gateway [In-Progress]
2. Plaid and Stripe sandbox integrations
3. Event journal, idempotency, and Python orchestration
4. Transactional outbox and Shippo integration
5. Retry handling and reconciliation
6. ClickHouse ETL and support integration
7. Observability, CI/CD, and an operator dashboard
8. Deployment documentation and reproducible demos