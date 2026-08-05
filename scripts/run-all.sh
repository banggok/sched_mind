#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
backend_pid=
frontend_pid=

cleanup() {
  trap - EXIT INT TERM

  if [ -n "$backend_pid" ]; then
    kill "$backend_pid" 2>/dev/null || true
  fi

  if [ -n "$frontend_pid" ]; then
    kill "$frontend_pid" 2>/dev/null || true
  fi

  if [ -n "$backend_pid" ]; then
    wait "$backend_pid" 2>/dev/null || true
  fi

  if [ -n "$frontend_pid" ]; then
    wait "$frontend_pid" 2>/dev/null || true
  fi
}

require_command() {
  command_name=$1

  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf '%s is unavailable. Run ./scripts/install.sh first.\n' "$command_name" >&2
    exit 1
  fi
}

wait_for_postgres() {
  container_id=$(docker compose ps -q postgres)
  if [ -z "$container_id" ]; then
    printf 'PostgreSQL container was not created.\n' >&2
    exit 1
  fi

  attempt=0
  while [ "$attempt" -lt 60 ]; do
    status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_id" 2>/dev/null || true)

    case "$status" in
      healthy)
        return 0
        ;;
      unhealthy | exited | dead)
        printf 'PostgreSQL failed to start. Recent container logs:\n' >&2
        docker compose logs --tail=50 postgres >&2 || true
        exit 1
        ;;
    esac

    attempt=$((attempt + 1))
    sleep 1
  done

  printf 'PostgreSQL did not become ready within 60 seconds. Recent container logs:\n' >&2
  docker compose logs --tail=50 postgres >&2 || true
  exit 1
}

trap cleanup EXIT
trap 'exit 130' INT TERM

if [ ! -f "$project_root/.env" ]; then
  printf '.env is missing. Run ./scripts/install.sh first.\n' >&2
  exit 1
fi

require_command docker
require_command go
require_command npm

if [ ! -d "$project_root/frontend/node_modules" ]; then
  printf 'Frontend dependencies are not installed. Run ./scripts/install.sh first.\n' >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  printf 'Docker Compose is unavailable. Update or reinstall Docker Desktop.\n' >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  printf 'Docker Desktop is not running. Start Docker Desktop and try again.\n' >&2
  exit 1
fi

cd "$project_root"

printf 'Starting the local PostgreSQL database...\n'
docker compose up -d postgres
wait_for_postgres
printf 'PostgreSQL is ready.\n'

"$project_root/scripts/run-backend.sh" &
backend_pid=$!

"$project_root/scripts/run-frontend.sh" &
frontend_pid=$!

printf '\nSchedMind is starting. Open:\n  http://localhost:5173\n\nPress Ctrl+C to stop the backend and frontend.\nThe PostgreSQL container remains running for the next session.\n\n'

while kill -0 "$backend_pid" 2>/dev/null &&
  kill -0 "$frontend_pid" 2>/dev/null; do
  sleep 1
done

set +e
if ! kill -0 "$backend_pid" 2>/dev/null; then
  wait "$backend_pid"
  exit_status=$?
else
  wait "$frontend_pid"
  exit_status=$?
fi
set -e

exit "$exit_status"
