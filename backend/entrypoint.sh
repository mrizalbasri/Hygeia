#!/bin/sh
set -e

# Run migrations once as a release step, using a direct connection.
# API replicas never migrate or seed the database on startup.
command="${1:-api}"
if [ "$#" -gt 0 ]; then shift; fi
case "$command" in
  api) exec /app/api "$@" ;;
  migrate)
    if [ -n "${DATABASE_URL_UNPOOLED:-}" ]; then
      export DATABASE_URL="$DATABASE_URL_UNPOOLED"
    fi
    case "${DATABASE_URL:-}" in
      *-pooler.*) echo "Migration requires a direct DATABASE_URL_UNPOOLED connection." >&2; exit 1 ;;
    esac
    exec /app/migrate "$@"
    ;;
  seed|seed-workflow|link-clerk-user) exec "/app/$command" "$@" ;;
  *) echo "Usage: api | migrate [up|version] | seed | seed-workflow | link-clerk-user [options]" >&2; exit 1 ;;
esac
