# xys-clone

仿小红书社交平台 —— Go + Gin + gRPC + RabbitMQ + Redis + MySQL + Vue

[![CI/CD Pipeline](https://github.com/ardi1s/open-the-bible/actions/workflows/ci-cd.yaml/badge.svg)](https://github.com/ardi1s/open-the-bible/actions/workflows/ci-cd.yaml)

## 项目结构

```
.
├── .github/workflows
│   ├── ci-cd.yaml         # CI/CD 流水线
│   └── README.md          # CI/CD 配置指南（Secrets、TCR 开通等）
├── cmd
│   ├── gateway            # Gin 网关入口（HTTP → gRPC）
│   └── user               # 用户服务入口
├── proto
│   └── user
│       └── user.proto     # 用户服务 Protobuf 定义
├── services
│   └── user               # 用户服务业务逻辑
├── docker-compose.yml
└── Makefile
```

## 环境要求

- Go 1.24+
- protoc（Protocol Buffers 编译器）
- Docker & Docker Compose（可选）

## 快速开始

### 1. 安装 protoc 插件（首次使用需执行）

```bash
make install-tools
```

### 2. 生成 gRPC 桩代码

```bash
make proto
```

### 3. 启动所有服务

```bash
# 方式 A：本地分别启动
make run-user     # 终端 1，监听 :50051
make run-gateway  # 终端 2，监听 :8080

# 方式 B：Docker Compose 一键启动
make docker-up
```

### 4. 验证服务

```bash
# 健康检查
curl http://localhost:8080/health
# → {"status":"ok"}

# 查询用户
curl http://localhost:8080/api/user/1
# → {"code":0,"data":{...}}

# gRPC 直连测试
grpcurl -plaintext -d '{"user_id":1}' localhost:50051 user.UserService/GetUser
```

## 可用命令

| 命令 | 说明 |
|------|------|
| `make install-tools` | 安装 protoc-gen-go / protoc-gen-go-grpc |
| `make proto` | 生成 proto 桩代码 |
| `make proto-clean` | 清理桩代码 |
| `make run-user` | 运行 user 服务 (:50051) |
| `make run-gateway` | 运行 gateway 服务 (:8080) |
| `make build-all` | 编译全部服务 |
| `make lint` | go vet 静态检查 |
| `make test` | 运行单元测试 |
| `make test-cover` | 运行测试 + 覆盖率报告 |
| `make ci-check` | lint + test（CI 本地预检） |
| `make docker-build` | 构建全部 Docker 镜像 |
| `make docker-push` | 推送全部 Docker 镜像 |
| `make docker-up` | 启动 docker-compose |
| `make docker-down` | 停止 docker-compose |

## CI/CD

流水线详情、新增服务、Secret 配置等操作指南见 [.github/workflows/README.md](.github/workflows/README.md)。

流水线三阶段：

```
Lint & Test ──▶ Build & Push ──▶ Deploy
 · go vet        · user 镜像       · kubectl 滚动更新
 · go test       · gateway 镜像    · 冒烟测试
```

| 事件 | 行为 |
|------|------|
| PR → main | Lint + Test |
| Push → main | Lint + Test → Build & Push → Deploy |
