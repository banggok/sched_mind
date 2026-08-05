#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
issues=

add_issue() {
  issues="${issues}
- $1"
}

version_at_least() {
  current_version=$1
  required_version=$2

  awk -v current="$current_version" -v required="$required_version" '
    function numeric_part(value) {
      sub(/[^0-9].*$/, "", value)
      return value == "" ? 0 : value + 0
    }
    BEGIN {
      split(current, current_parts, ".")
      split(required, required_parts, ".")

      for (part_index = 1; part_index <= 3; part_index++) {
        current_part = numeric_part(current_parts[part_index])
        required_part = numeric_part(required_parts[part_index])

        if (current_part > required_part) {
          exit 0
        }
        if (current_part < required_part) {
          exit 1
        }
      }

      exit 0
    }
  '
}

if ! command -v git >/dev/null 2>&1; then
  add_issue "Git is not installed."
fi

if ! command -v go >/dev/null 2>&1; then
  add_issue "Go 1.24 or later is not installed."
else
  go_version=$(go env GOVERSION 2>/dev/null | sed 's/^go//')
  if ! version_at_least "$go_version" "1.24.0"; then
    add_issue "Go 1.24 or later is required; found $go_version."
  fi
fi

if ! command -v node >/dev/null 2>&1; then
  add_issue "Node.js 20.19 or later is not installed."
else
  node_version=$(node --version 2>/dev/null | sed 's/^v//')
  if ! version_at_least "$node_version" "20.19.0"; then
    add_issue "Node.js 20.19 or later is required; found $node_version."
  fi
fi

if ! command -v npm >/dev/null 2>&1; then
  add_issue "npm 10 or later is not installed."
else
  npm_version=$(npm --version 2>/dev/null)
  if ! version_at_least "$npm_version" "10.0.0"; then
    add_issue "npm 10 or later is required; found $npm_version."
  fi
fi

if ! command -v docker >/dev/null 2>&1; then
  add_issue "Docker Desktop is not installed."
elif ! docker compose version >/dev/null 2>&1; then
  add_issue "Docker Compose is unavailable. Update or reinstall Docker Desktop."
elif ! docker info >/dev/null 2>&1; then
  add_issue "Docker Desktop is installed but not running."
fi

if [ -n "$issues" ]; then
  printf 'SchedMind installation cannot continue.\n\nResolve these requirements:%s\n\nThen run ./scripts/install.sh again.\n' "$issues" >&2
  exit 1
fi

cd "$project_root"

if [ ! -f .env ]; then
  cp .env.example .env
  printf 'Created .env from .env.example.\n'
else
  printf 'Using existing .env.\n'
fi

printf '\nDownloading backend dependencies...\n'
(
  cd backend
  go mod download
)

printf '\nInstalling frontend dependencies...\n'
(
  cd frontend
  npm ci
)

printf '\nStarting the local PostgreSQL database...\n'
docker compose up -d postgres

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
      printf '\nSchedMind installation is complete.\n\nStart the application with:\n  ./scripts/run-all.sh\n'
      exit 0
      ;;
    unhealthy | exited | dead)
      printf '\nPostgreSQL failed to start. Recent container logs:\n' >&2
      docker compose logs --tail=50 postgres >&2 || true
      exit 1
      ;;
  esac

  attempt=$((attempt + 1))
  sleep 1
done

printf '\nPostgreSQL did not become ready within 60 seconds. Recent container logs:\n' >&2
docker compose logs --tail=50 postgres >&2 || true
exit 1
