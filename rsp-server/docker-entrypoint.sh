#!/bin/sh
# Apply golang-migrate then exec the API. Render/Docker Linux only.
# RUN_MIGRATIONS=0 skips migrate (operator ran make migrate-up).
# DATABASE_URL_MIGRATE, if set, is the migrate DSN (Supabase direct :5432).
# DATABASE_URL is the runtime DSN (pooler :6543 is fine for the app).
set -e
dsn=${DATABASE_URL_MIGRATE:-$DATABASE_URL}
if [ "${RUN_MIGRATIONS:-1}" != "0" ] && [ -n "$dsn" ]; then
	echo "rsp: migrate up"
	/usr/local/bin/migrate -path /migrations -database "$dsn" up
fi
exec /server
