# xys-clone

仿小红书社交平台 —— Go + Gin + gRPC + RabbitMQ + Redis + MySQL + Vue

[![CI/CD Pipeline](https://github.com/ardi1s/open-the-bible/actions/workflows/ci-cd.yaml/badge.svg)](https://github.com/ardi1s/open-the-bible/actions/workflows/ci-cd.yaml)

## 项目结构

```
.
├── .github/workflows
│   └── ci-cd.yaml         # CI/CD 流水线
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

### 流水线触发条件

| 事件 | 触发 | 说明 |
|------|------|------|
| PR → main | ✅ | Lint + Test 通过才允许合入 |
| Push → main | ✅ | Lint + Test → Build & Push → Deploy |

### 流水线阶段

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│  Job 1       │     │  Job 2          │     │  Job 3       │
│  Lint & Test │ ──▶ │  Build & Push   │ ──▶ │  Deploy      │
│  · go vet    │     │  · user 镜像    │     │  · kubectl   │
│  · go test   │     │  · gateway 镜像 │     │  · 冒烟测试  │
└──────────────┘     └─────────────────┘     └──────────────┘
```

### 新增服务

只需两处修改（后续 note / feed / rank 服务同理）：

**1. 服务清单** — `.github/workflows/ci-cd.yaml` 第 21 行：
```yaml
SERVICES: '["user","gateway","note"]'   # ← 追加新服务名
```

**2. Makefile** — 第 18 行：
```makefile
SERVICES := user gateway note   # ← 追加新服务名
```

**3. 约定**：创建 `cmd/<name>/Dockerfile` 和 `cmd/<name>/main.go`，构建会自动生效。

### 配置 GitHub Secrets

在仓库 `Settings → Secrets and variables → Actions → New repository secret` 中添加：

#### 必填

| Secret | 值示例 | 说明 |
|--------|--------|------|
| `DOCKER_USERNAME` | `yourname` | Docker Hub / ACR 用户名 |
| `DOCKER_PASSWORD` | `dckr_pat_xxx...` | Docker Hub Access Token（**不要用密码**，在 Hub → Account Settings → Security → New Access Token 创建） |

#### 选填（K8s 接入后再配）

| Secret | 值示例 | 说明 |
|--------|--------|------|
| `KUBE_CONFIG` | `cat ~/.kube/config \| base64` | Base64 编码的 kubeconfig |

### 触发第一次 CI/CD

```bash
# 1. 初始化 Git（如果还没做）
git init && git add . && git commit -m "feat: init xys-clone"

# 2. 创建 GitHub 仓库，关联 remote
git remote add origin git@github.com:<OWNER>/<REPO>.git

# 3. 创建 main 分支并推送
git branch -M main
git push -u origin main
```

推送后打开 GitHub 仓库 → **Actions** 标签页，即可看到流水线实时运行。

### 本地预检（推送前）

```bash
make ci-check   # 相当于 CI Job 1，本地跑一遍放心
```
