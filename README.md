# Asset Management System

## Overview

A secure Go REST API for registering assets, assigning them to users, returning them, placing them in maintenance, retiring them, and reviewing immutable custody history.

## Features

- Revocable JWT login with admin, custodian, and employee roles
- Asset catalog with tags, serial numbers, categories, dates, costs, search, filters, and pagination
- Atomic assignment, return, maintenance, availability, and retirement transitions
- Immutable asset-event history with actor and timestamp
- Employee ownership filtering for asset records and history
- Role-aware dashboard counts and asset value
- PostgreSQL migrations, seed accounts, tests, OpenAPI, Docker, and CI

## Technology Stack

Go 1.25+, Chi, pgx, PostgreSQL 14+, bcrypt, JWT, Docker Compose.

## Requirements

Go 1.25+, PostgreSQL 14+, and optionally Docker with Compose.

## Installation

```bash
git clone https://github.com/Arjunren/asset-management-system.git
cd asset-management-system
go mod download
cp .env.example .env
```

## Environment Variables

Configure `APP_ADDR`, `DATABASE_URL`, `JWT_SECRET`, `JWT_EXPIRY`, `ALLOWED_ORIGINS`, and the `SEED_*` values shown in `.env.example`. Sample credentials are for local development only.

## Database Setup

```bash
go run ./cmd/migrate
go run ./cmd/seed
```

## Running the Application

```bash
go run ./cmd/api
```

Docker:

```bash
docker compose up -d db
docker compose --profile tools run --rm migrate
docker compose --profile tools run --rm seed
docker compose up -d api
```

## Running Tests

```bash
go test ./...
go vet ./...
```

CI additionally runs the race detector, a live PostgreSQL concurrent-assignment test, and all builds.

## Default Development Accounts

- Admin: `admin@example.com` / `AdminPass123!`
- Custodian: `custodian@example.com` / `Custodian123!`
- Employee: `employee@example.com` / `Employee123!`

## API Endpoints

`POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/dashboard`, admin-only `POST /api/users`, `GET|POST /api/assets`, `GET|PUT /api/assets/{id}`, `POST /api/assets/{id}/transition`, and `GET /api/assets/{id}/events`. Lists support `page`, `limit`, `search`, `status`, and `category`. See `docs/openapi.yaml`.

## Folder Structure

`cmd/` contains entry points, `internal/domain` models, `internal/service` validation and ownership, `internal/store` interfaces, `internal/postgres` transactional persistence, `internal/httpapi` transport and authorization, `migrations/` schema, and `docs/` API documentation.

## Security Notes

Passwords are bcrypt-hashed and excluded from responses. JWTs require a live database session. Employee reads are filtered and individually ownership-checked. Server middleware enforces admin and custodian operations. Asset transitions lock the asset row, validate its current state, update custody, and append history in one transaction. DTOs reject unknown fields and oversized bodies; SQL is parameterized; CORS is allowlisted; login is rate-limited; errors are sanitized.

## Known Limitations

- API-only with no barcode scanning or file uploads.
- Straight-line depreciation, procurement approval, and vendor management are outside this small scope.
- Purchase cost is stored as PostgreSQL numeric but represented as a JSON number.
