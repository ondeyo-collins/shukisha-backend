#!/usr/bin/env sh
set -eu

if [ ! -f ".env.docker" ]; then
  echo "Missing .env.docker. Create it with:"
  echo "  cp .env.docker.example .env.docker"
  exit 1
fi

exec docker compose --env-file .env.docker "$@"

