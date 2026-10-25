#!/bin/sh
set -e

echo "==> Running migrations..."
./migrate up

if [ "$APP_ENV" = "development" ] || [ "$APP_ENV" = "test" ]; then
  echo "==> Seeding demo data..."
  ./seed
fi

echo "==> Starting Hygeia API on :${PORT:-8080}"
exec ./api
