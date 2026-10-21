# Keep in sync with internal/redact. Used by docker-entrypoint.sh.
s#postgres://[^[:space:]"']*@#postgres://***@#g
s#postgresql://[^[:space:]"']*@#postgresql://***@#g
s#[Pp]assword=[^[:space:]"']*#password=***#g
s#[Ff]or user "[^"]*"#for user "***"#g
s#[Ff]or user '[^']*'#for user '***'#g
