# ──────────────────────────────────────────────
# User 服务 Dockerfile
#   阶段一：Go 编译（golang:1.25-alpine）
#   阶段二：最小运行镜像（alpine:3.20），仅包含二进制 + CA 证书
# ──────────────────────────────────────────────

# ── 构建阶段 ──
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 先复制依赖文件，利用 Docker 缓存层加速
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码并编译
COPY . .
RUN go build -o /user-service ./cmd/user

# ── 运行阶段 ──
FROM alpine:3.20

# 安装 CA 证书（HTTPS 调用需要）
RUN apk add --no-cache ca-certificates

COPY --from=builder /user-service /user-service

EXPOSE 50051

ENTRYPOINT ["/user-service"]
