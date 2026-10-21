# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM migrate/migrate:v4.18.1 AS migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget \
	&& adduser -D -H -u 65532 app
COPY --from=build /out/server /server
COPY --from=migrate /usr/local/bin/migrate /usr/local/bin/migrate
COPY db/migrations /migrations
COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh /usr/local/bin/migrate
USER 65532
EXPOSE 8080
ENV RUN_MIGRATIONS=1
ENTRYPOINT ["/entrypoint.sh"]
