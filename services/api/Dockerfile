# ── Atlas stage ───────────────────────────────────────────────────────────────
ARG ATLAS_VERSION=1.2.0
FROM arigaio/atlas:${ATLAS_VERSION} AS atlas

# ── Dev stage ─────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS dev

RUN apk add --no-cache git build-base && \
    go install github.com/air-verse/air@latest && \
    go install github.com/swaggo/swag/cmd/swag@v1.16.6

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY --from=atlas /atlas /usr/local/bin/atlas
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["air"]

# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/tachigo ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 65532 nonroot && \
    adduser -S -D -H -u 65532 -G nonroot nonroot && \
    mkdir -p /home/nonroot && \
    chown nonroot:nonroot /home/nonroot

WORKDIR /app

COPY --from=atlas /atlas /usr/local/bin/atlas
COPY --from=builder /app/bin/tachigo /tachigo
COPY --from=builder /app/migrations ./migrations
COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/tachigo"]
