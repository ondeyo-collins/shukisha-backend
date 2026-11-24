# Shukisha Backend

The API and background worker behind Shukisha, a parcel delivery and logistics platform.

## Features

- **Parcels:** booking, payment callbacks, end-to-end tracking, exceptions, and proof of delivery
- **Drivers:** OTP onboarding, live sessions with heartbeats, parcel offers, and stop requests
- **Routes and fleet:** route and stop management, a live fleet view, and stop alerts
- **Finance:** ledger, unpaid balances, and payout execution
- **Admin:** role-based access by department (Logistics, Transport, Finance, Support), audit logs, claims with attachments, notifications, and dashboard stats

## Stack

Go, GORM on PostgreSQL, Redis, and JWT auth.

| Path | Purpose |
| --- | --- |
| `cmd/api` | HTTP API server |
| `cmd/worker` | Background jobs |
| `internal/` | Domain modules (admin, drivers, parcels, routes, ledger, ...) |
| `migrations/` | SQL schema |
| `scripts/` | Seed and maintenance scripts |
| `docs/openapi.yaml` | API spec, also served at `/v1/openapi.yaml` |

## Getting started

See [DEV_START.md](DEV_START.md) for local setup and [README_DEPLOY.md](README_DEPLOY.md) for deployment.

Seed scripts read the admin password from the `SEED_ADMIN_PASSWORD` environment variable.
