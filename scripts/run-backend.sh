#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

if [ -f "$project_root/.env" ]; then
  set -a
  # shellcheck disable=SC1091
  . "$project_root/.env"
  set +a
fi

cd "$project_root/backend"
exec go run ./cmd/api
