#!/bin/sh
# Runs once on first Postgres boot (empty data volume).
# Applies goose-formatted migrations, stripping the Down section so we
# don't create-then-drop the schema in the same init pass.
set -eu

for f in /migrations/*.sql; do
	[ -f "$f" ] || continue
	echo "applying $(basename "$f") ..."
	# Keep everything up to (but not including) the goose Down marker.
	sed '/^-- +goose Down/,$d' "$f" | psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"
done

echo "migrations applied."
