# ──────────────────────────────────────────────
# Interaction 服务 Dockerfile
# ──────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /interaction-service ./cmd/interaction

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /interaction-service /interaction-service
EXPOSE 50054
ENTRYPOINT ["/interaction-service"]
