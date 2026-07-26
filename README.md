# SchedMind

> Smarter Planning. Better Delivery.

Minimal foundation for a scheduling application, with a Go backend organized
using Domain-Driven Design boundaries and a React/Tailwind frontend.

## Requirements

- Go 1.24 or later
- Node.js 20.19 or later
- npm 10 or later
- Docker Desktop

## Local database

SchedMind uses PostgreSQL 16 in Docker when the application is running locally.
SQLite is only used as an isolated database by automated backend tests.

Create a local environment file once:

```sh
cp .env.example .env
```

The checked-in `.env.example` contains safe local-development values. Change
`.env` when your local ports or credentials differ; production must provide its
own environment variables.

Start PostgreSQL before running the application:

```sh
docker compose up -d postgres
```

The example configuration connects the backend to:

```text
postgres://schedmind:schedmind@localhost:5432/schedmind?sslmode=disable
```

`DATABASE_URL`, `HTTP_ADDRESS`, and `HTTP_SHUTDOWN_TIMEOUT` are required by the
backend. Database migrations run automatically when the API starts. On
`SIGINT` or `SIGTERM`, the API stops accepting new requests, waits for active
requests up to `HTTP_SHUTDOWN_TIMEOUT`, and closes its database connection.

The frontend uses `VITE_API_BASE_URL` for browser requests and
`VITE_BACKEND_PROXY_TARGET` for the local Vite development proxy.

To stop the database without deleting its data:

```sh
docker compose stop postgres
```

## Run the backend

```sh
./scripts/run-backend.sh
```

The API listens on `http://localhost:8080`. Its health endpoints are available
at `GET /health` and `GET /api/health`.

Capacity Override endpoints are nested under Members:

```text
GET    /api/team-members/{teamMemberId}/capacity-overrides
GET    /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}
POST   /api/team-members/{teamMemberId}/capacity-overrides
PUT    /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}
DELETE /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}
```

Capacity Override create and update payloads require `description` (maximum 100
characters) together with `startDate`, `endDate`, and `capacity`.

The list accepts `page`, `pageSize`, and an optional date-only `effectiveDate`:

```text
GET /api/team-members/{teamMemberId}/capacity-overrides?effectiveDate=2026-07-27&page=1&pageSize=5
```

Filtering is inclusive (`startDate <= effectiveDate <= endDate`). The default
page size is `5`; invalid Effective Date values return
`400 INVALID_EFFECTIVE_DATE`.

Public Holiday endpoints are global Schedule resources:

```text
GET    /api/public-holidays
GET    /api/public-holidays/{publicHolidayId}
GET    /api/public-holidays/calendar?startDate=2026-08-01&endDate=2026-08-31
POST   /api/public-holidays
PUT    /api/public-holidays/{publicHolidayId}
DELETE /api/public-holidays/{publicHolidayId}
```

The list accepts `page`, `pageSize`, and optional exact date-only
`holidayDate`. Create and update require inclusive date-only `startDate` and
`endDate` plus a trimmed `description` of at most 100 characters. Weekend dates
are skipped because Saturday and Sunday are holidays by default. Each weekday
may belong to only one Public Holiday range; any conflict rejects the complete
mutation. Active and upcoming ranges are listed before expired ranges using the
configured `APP_TIMEZONE`.

Project endpoints:

```text
GET    /api/projects?search=alpha&page=1&pageSize=5
GET    /api/projects/{projectId}
POST   /api/projects
PUT    /api/projects/{projectId}
POST   /api/projects/{projectId}/status
POST   /api/projects/{projectId}/priority
DELETE /api/projects/{projectId}
```

## Run the frontend

Install frontend dependencies once:

```sh
cd frontend
npm install
```

Then start the development server:

```sh
./scripts/run-frontend.sh
```

## Run backend and frontend together

After installing the frontend dependencies, start both development servers with:

```sh
./scripts/run-all.sh
```

Press `Ctrl+C` to stop both servers. If either server exits, the script stops
the other server and returns the exit status of the server that exited first.

## Validate

```sh
cd backend
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
```

```sh
cd frontend
npm run format:check
npm run lint
npm run typecheck
npm test
npm run build
```
