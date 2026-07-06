# ──────────────────────────────────────────────
# Rank 服务 Dockerfile
# ──────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /rank-service ./cmd/rank

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /rank-service /rank-service
EXPOSE 50055
ENTRYPOINT ["/rank-service"]
