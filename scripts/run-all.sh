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

  wait "$backend_pid" 2>/dev/null || true
  wait "$frontend_pid" 2>/dev/null || true
}

trap cleanup EXIT
trap 'exit 130' INT TERM

"$project_root/scripts/run-backend.sh" &
backend_pid=$!

"$project_root/scripts/run-frontend.sh" &
frontend_pid=$!

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

