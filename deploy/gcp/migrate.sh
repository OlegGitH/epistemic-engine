#!/usr/bin/env sh
set -eu
: "${DATABASE_URL:?DATABASE_URL is required}"
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c 'CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())'
for migration in /migrations/*.up.sql; do
  version="$(basename "$migration" .up.sql)"
  if [ "$(psql "$DATABASE_URL" -Atc "SELECT count(*) FROM schema_migrations WHERE version = '${version}'")" = "1" ]; then
    echo "already applied $(basename "$migration")"
    continue
  fi
  echo "applying $(basename "$migration")"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(version) VALUES ('${version}')"
done
