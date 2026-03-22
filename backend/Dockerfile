# ── Dev stage ─────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS dev

RUN apk add --no-cache git build-base && \
    go install github.com/air-verse/air@latest && \
    go install github.com/swaggo/swag/cmd/swag@v1.8.12

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

EXPOSE 8080

CMD ["air"]

# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/tachigo ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
# distroless/static includes ca-certificates but not tzdata.
# Copy the timezone database from the builder so time.LoadLocation works.
FROM gcr.io/distroless/static-nonroot:nonroot

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/tachigo /tachigo

EXPOSE 8080

ENTRYPOINT ["/tachigo"]
