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
npm test
npm run lint
npm run build
```
