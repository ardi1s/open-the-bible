# xys-clone

仿小红书社交平台 —— Go + Gin + gRPC + RabbitMQ + Redis + MySQL + Vue

[![CI/CD Pipeline](https://github.com/ardi1s/open-the-bible/actions/workflows/ci-cd.yaml/badge.svg)](https://github.com/ardi1s/open-the-bible/actions/workflows/ci-cd.yaml)

## 项目结构

```
.
├── .github/workflows
│   ├── ci-cd.yaml         # CI/CD 流水线
│   └── README.md          # CI/CD 配置指南
├── cmd
│   ├── gateway            # Gin 网关入口（HTTP → gRPC）
│   ├── note               # 笔记服务入口
│   └── user               # 用户服务入口
├── proto
│   ├── note               # 笔记服务 proto
│   └── user               # 用户服务 proto
├── services
│   ├── note               # 笔记服务业务逻辑（MySQL + RabbitMQ）
│   └── user               # 用户服务业务逻辑
├── db
│   └── init.sql           # 数据库初始化脚本
├── docker-compose.yml
└── Makefile
```

## 环境要求

- Go 1.25+
- protoc（Protocol Buffers 编译器）
- Docker & Docker Compose（可选）
- MySQL 8.0（brew install mysql）
- RabbitMQ（brew install rabbitmq）

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
# 方式 A：Docker Compose 一键启动（推荐，含 MySQL + RabbitMQ）
make docker-up

# 方式 B：本地分别启动
#   需要先在本机启动 MySQL（端口 3306）和 RabbitMQ（端口 5672）
#   并执行 db/init.sql 初始化库表
make run-user     # 终端 1，:50051
make run-note     # 终端 2，:50052
make run-gateway  # 终端 3，:8080
```

### 4. 验证服务

```bash
# 健康检查
curl http://localhost:8080/health
# → {"status":"ok"}

# 查询用户
curl http://localhost:8080/api/user/1
# → {"code":0,"data":{...}}

# 创建笔记
curl -X POST http://localhost:8080/api/notes \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"title":"你好世界","content":"这是我的第一篇笔记","image_urls":["a.jpg"],"tags":["生活"]}'
# → {"code":0,"data":{"note_id":1}}

# gRPC 直连测试
grpcurl -plaintext -d '{"user_id":1}' localhost:50051 user.UserService/GetUser
grpcurl -plaintext -d '{"user_id":1,"title":"hi","content":"hello"}' localhost:50052 note.NoteService/CreateNote
```

## 可用命令

| 命令 | 说明 |
|------|------|
| `make install-tools` | 安装 protoc-gen-go / protoc-gen-go-grpc |
| `make proto` | 生成所有 proto 桩代码 |
| `make proto-clean` | 清理桩代码 |
| `make run-user` | 运行 user 服务 (:50051) |
| `make run-note` | 运行 note 服务 (:50052) |
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

## 基础设施端口

| 服务 | 端口 | 说明 |
|------|------|------|
| gateway | 8080 | Gin HTTP 网关 |
| user | 50051 | gRPC 用户服务 |
| note | 50052 | gRPC 笔记服务 |
| MySQL | 3306 | 数据库 |
| RabbitMQ | 5672 | AMQP 消息队列 |
| RabbitMQ 管理界面 | 15672 | 网页管理（guest/guest） |

## 验证 RabbitMQ 消息

创建笔记后，进入 RabbitMQ 管理界面验证：

```bash
# Docker Compose 启动后，浏览器打开
open http://localhost:15672
```

登录（guest / guest）后：
1. 点击 **Exchanges** → 找到 `note.events`
2. 创建笔记后回到 **Queues**，点击 **Get messages** 即可查看投递的消息

或命令行验证（需进入容器）：

```bash
docker exec xys-rabbitmq rabbitmqctl list_queues
```

## CI/CD

流水线详情、新增服务、Secret 配置等操作指南见 [.github/workflows/README.md](.github/workflows/README.md)。

```
Lint & Test ──▶ Build & Push ──▶ Deploy
 · go vet        · user 镜像       · kubectl 滚动更新
 · go test       · gateway 镜像    · 冒烟测试
                 · note 镜像
```

| 事件 | 行为 |
|------|------|
| PR → main | Lint + Test |
| Push → main | Lint + Test → Build & Push → Deploy |
