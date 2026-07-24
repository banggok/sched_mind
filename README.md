# SchedMind

> Smarter Planning. Better Delivery.

Minimal foundation for a scheduling application, with a Go backend organized
using Domain-Driven Design boundaries and a React/Tailwind frontend.

## Requirements

- Go 1.24 or later
- Node.js 20.19 or later
- npm 10 or later

## Run the backend

```sh
./scripts/run-backend.sh
```

The API listens on `http://localhost:8080`. Its health endpoint is available at
`GET /health`.

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
