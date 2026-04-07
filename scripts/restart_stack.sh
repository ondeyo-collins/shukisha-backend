#!/usr/bin/env sh
set -eu

# Restart Shukisha backend stack using .env.docker consistently.
#
# Usage:
#   ./scripts/restart_stack.sh
#   ./scripts/restart_stack.sh --wipe   # also deletes volumes (DATA LOSS)

if [ ! -f "./compose-docker.sh" ]; then
  echo "Run from the backend repo root (where compose-docker.sh exists)."
  exit 1
fi

wipe="false"
if [ "${1:-}" = "--wipe" ]; then
  wipe="true"
fi

chmod +x ./compose-docker.sh || true

echo "Stopping containers..."
if [ "$wipe" = "true" ]; then
  echo "WARNING: wiping volumes (postgres_data, redis_data)"
  ./compose-docker.sh down -v
else
  ./compose-docker.sh down
fi

echo "Starting stack (build if needed)..."
./compose-docker.sh up -d --build

echo ""
echo "Status:"
./compose-docker.sh ps

echo ""
echo "Health:"
curl -fsS http://localhost:8080/health || true
echo ""

echo "Recent logs (api):"
./compose-docker.sh logs -n 60 api || true

