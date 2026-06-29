# ──────────────────────────────────────────────
# Gateway 服务 Dockerfile
#   阶段一：Go 编译（golang:1.24-alpine）
#   阶段二：最小运行镜像（alpine:3.20）
#   通过环境变量 USER_SERVICE_ADDR 指定下游 user 地址
# ──────────────────────────────────────────────

# ── 构建阶段 ──
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 先复制依赖文件，利用 Docker 缓存层加速
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码并编译
COPY . .
RUN go build -o /gateway ./cmd/gateway

# ── 运行阶段 ──
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /gateway /gateway

EXPOSE 8080

ENTRYPOINT ["/gateway"]
