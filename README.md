# CRM-Agentic-Flow

`CRM-Agentic-Flow` is a multi-tenant CRM focused on two things:

- a usable sales workspace for accounts, contacts, deals, tasks, and notes
- an operational backbone with persistent outbox events, webhook delivery tracking, replay, and auditability

The current direction is an initial product somewhere between `Pipedrive` and `HubSpot-lite`, with stronger backend observability from day one.

## Current Scope

What is already implemented:

- tenant-aware authentication
- core CRM resources:
  - `organizations`
  - `people`
  - `deals`
  - `tasks`
  - `notes`
- persistent `outbox_events`
- PostgreSQL-backed event processor
- webhook endpoints, subscriptions, and deliveries
- manual replay for outbox events and deliveries
- audit log
- operational stats for outbox and webhook deliveries
- browser UI served by the API
- smoke test and CI workflow

## Architecture

High-level flow:

```text
API -> PostgreSQL (source of truth + outbox_events) -> event-processor -> webhook_deliveries -> external endpoints
```

Main components:

- `cmd/api`
  - HTTP API
  - embedded UI
  - auth
  - CRM resources
  - outbox/audit/ops endpoints
- `cmd/event-processor`
  - claims pending outbox events
  - delivers subscribed webhooks
  - manages retries and status transitions
- `db/init`
  - schema and seed SQL
- `scripts`
  - smoke test pipeline
  - local webhook receiver

## Quick Start

Requirements:

- Go
- Docker + Docker Compose

Start the database:

```bash
make db-up
```

Run the API:

```bash
make api
```

Run the event processor in another terminal:

```bash
make worker
```

Run the end-to-end smoke test:

```bash
make smoke
```

Base URLs:

- API: `http://localhost:8080`
- Database: `postgresql://postgres:postgres@localhost:5440/crm_agentic_flow`

Development seed credentials:

- email: `admin@crmflow.local`
- password: `admin123`
- tenant slug: `demo`

## UI

The API serves a built-in browser UI at:

```text
http://localhost:8080/app
```

The UI currently includes:

- dashboard and operational summary
- organizations, people, deals, and tasks
- organization detail with related entities
- deal pipeline view
- outbox and webhook monitoring
- replay actions from the interface

## Repo Layout

```text
cmd/
  api/
  event-processor/
  migration-agent/
db/
  init/
domain/
scripts/
```

## Operations

Operational usage, troubleshooting, replay flows, and webhook management are documented in [OPERATIONS.md](./OPERATIONS.md).

## Testing

Run API tests:

```bash
make test-api
```

Run the local CI flow:

```bash
make ci
```

The CI workflow also runs the smoke test pipeline.

## Status

This is an active build, not a finished product. The current priority is hardening the core CRM + ops backbone before moving deeper into automation, reporting, and AI-assisted workflows.
