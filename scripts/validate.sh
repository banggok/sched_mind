#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

run_step() {
  label=$1
  shift

  printf '\n==> %s\n' "$label"
  "$@" || {
    status=$?
    printf '\nFAILED: %s (exit code %s)\n' "$label" "$status" >&2
    exit "$status"
  }
}

cd "$project_root/backend"
run_step "Backend: go fmt ./..." go fmt ./...
run_step "Backend: go vet ./..." go vet ./...
run_step "Backend: go test ./..." go test ./...
run_step "Backend: go test -race ./..." go test -race ./...

cd "$project_root/frontend"
run_step "Frontend: npx prettier --write ." npx prettier --write .
run_step "Frontend: npm run format:check" npm run format:check
run_step "Frontend: npm run lint" npm run lint
run_step "Frontend: npm run typecheck" npm run typecheck
run_step "Frontend: npm test" npm test
run_step "Frontend: npm run build" npm run build

printf '\nAll validation steps passed.\n'
