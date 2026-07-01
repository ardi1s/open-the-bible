# ──────────────────────────────────────────────
# Feed 服务 Dockerfile
#   阶段一：Go 编译（golang:1.25-alpine）
#   阶段二：最小运行镜像（alpine:3.20）
#   通过环境变量连接 MySQL / Redis / RabbitMQ / gRPC 下游
# ──────────────────────────────────────────────

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /feed-service ./cmd/feed

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /feed-service /feed-service

EXPOSE 50053

ENTRYPOINT ["/feed-service"]
