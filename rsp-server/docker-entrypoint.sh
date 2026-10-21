#!/bin/sh
# Apply golang-migrate then exec the API. Render/Docker Linux only.
# RUN_MIGRATIONS=0 skips migrate (operator ran make migrate-up).
# DATABASE_URL_MIGRATE, if set, is the migrate DSN (Supabase direct :5432).
# DATABASE_URL is the runtime DSN (pooler :6543 is fine for the app).
#
# Do not `set -x`: it would print DATABASE_URL. golang-migrate embeds the
# DSN (password and all) in errors like: failed to open database, "<DSN>".
# Filter that before it hits Render logs.
set -e

redact_log() {
	sed -e "s#postgres://[^[:space:]\"']*@#postgres://***@#g" \
		-e "s#postgresql://[^[:space:]\"']*@#postgresql://***@#g" \
		-e "s#[Pp]assword=[^[:space:]\"']*#password=***#g"
}

dsn=${DATABASE_URL_MIGRATE:-$DATABASE_URL}
if [ "${RUN_MIGRATIONS:-1}" != "0" ] && [ -n "$dsn" ]; then
	echo "rsp: migrate up"
	set +e
	out=$(/usr/local/bin/migrate -path /migrations -database "$dsn" up 2>&1)
	st=$?
	set -e
	printf '%s\n' "$out" | redact_log
	if [ "$st" -ne 0 ]; then
		echo "rsp: migrate failed" >&2
		exit "$st"
	fi
fi
exec /server
