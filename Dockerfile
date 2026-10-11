# =====================
# BUILD STAGE
# =====================
FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# VERSION is written by the deploy job (the commit SHA); /health reports it.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X abdanhafidz.com/event-booking-api/router.Version=$(cat VERSION 2>/dev/null || echo dev)" \
    -o app . \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o seed ./cmd/seed

# =====================
# RUNTIME STAGE
# =====================
# Alpine rather than distroless: the entrypoint is a shell script.
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 1000 app

WORKDIR /app

COPY --from=builder /app/app /app/seed /app/
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Hugging Face Spaces routes traffic to 7860 by default.
ENV HOST_ADDRESS=0.0.0.0 \
    HOST_PORT=7860 \
    GIN_MODE=release \
    SEED_ON_STARTUP=true

# Non-root user (uid 1000, as Hugging Face Spaces expects)
USER app

EXPOSE 7860

ENTRYPOINT ["/app/entrypoint.sh"]
