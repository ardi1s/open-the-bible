# CI/CD 配置指南

本文档对应 `.github/workflows/ci-cd.yaml` 流水线的配置与运维说明。

## 流水线概览

```
push / PR → main
      │
      ▼
┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│  Job 1       │     │  Job 2          │     │  Job 3       │
│  Lint & Test │ ──▶ │  Build & Push   │ ──▶ │  Deploy      │
│  · go vet    │     │  · user 镜像    │     │  · kubectl   │
│  · go test   │     │  · gateway 镜像 │     │  · 冒烟测试  │
└──────────────┘     └─────────────────┘     └──────────────┘
```

| 事件 | Job 1 | Job 2 | Job 3 |
|------|:-----:|:-----:|:-----:|
| PR → main | ✅ | — | — |
| Push → main | ✅ | ✅ | ✅ |

---

## 新增服务

后续添加 note / feed / rank 等服务时，只需改两处：

**1.** `.github/workflows/ci-cd.yaml` — `SERVICES` 数组追加名字：

```yaml
SERVICES: '["user","gateway","note"]'
```

**2.** `Makefile` — `SERVICES` 列表追加名字：

```makefile
SERVICES := user gateway note
```

**3.** 约定：创建 `cmd/<name>/Dockerfile` 和 `cmd/<name>/main.go`。CI 构建脚本会遍历 `SERVICES`，自动发现并构建对应 Dockerfile。

---

## 配置 GitHub Secrets

使用 Docker Hub 作为镜像仓库。由于你可能是通过 GitHub 授权登录 Docker Hub 的，没有传统密码，所以需要用 Access Token 代替。

### 第一步：创建 Docker Hub Access Token

> ⚠️ 就算你网页端是用 GitHub 登录的，命令行 `docker login` 也不认 GitHub 授权，必须用 Access Token。

1. 打开 [Docker Hub → Account Settings → Security](https://hub.docker.com/settings/security)
2. 点击 **New Access Token**
3. 随便起个名（比如 `xys-clone-ci`），权限选 **Read & Write**
4. 点 Generate，复制生成的 token（格式：`dckr_pat_xxxxxxxxxxxxxxxxx`）
5. **立即保存**，关闭后无法再次查看

### 第二步：确认你的 Docker Hub 用户名

右上角头像旁边那个就是，或者 Settings 页面顶部显示的 `@username`。

### 第三步：填入 GitHub Secrets

打开仓库 → **Settings → Secrets and variables → Actions → New repository secret**，添加：

| Secret | 填什么 |
|--------|--------|
| `DOCKER_USERNAME` | 你的 Docker Hub 用户名 |
| `DOCKER_PASSWORD` | 第一步生成的 Access Token（`dckr_pat_...`） |

---

### 对应关系

```
┌──────────────────────────────────────────────────────────┐
│                     GitHub Secrets                        │
│  DOCKER_USERNAME = ardi1s         ──▶ docker login 用户名 │
│  DOCKER_PASSWORD = dckr_pat_xxx  ──▶ docker login 密码   │
└──────────────────────────────────────────────────────────┘
                           │
                           ▼
                 docker login docker.io
                           │
                           ▼
          docker push docker.io/ardi1s/xys-user:latest
```

### 镜像最终路径

```
docker.io/<你的用户名>/xys-user:latest
docker.io/<你的用户名>/xys-user:<git-sha前7位>
docker.io/<你的用户名>/xys-gateway:latest
docker.io/<你的用户名>/xys-gateway:<git-sha前7位>
```

---

### 选填（K8s 接入后再配）

| Secret | 值示例 | 说明 |
|--------|--------|------|
| `KUBE_CONFIG` | `cat ~/.kube/config \| base64` | Base64 编码的 kubeconfig |

---

## 触发第一次 CI/CD

```bash
git add .
git commit -m "feat: ..."
git push origin main
```

推送后打开 GitHub 仓库 → **Actions** 标签页查看运行结果。

> Job 1（Lint & Test）无需任何配置即可通过。Job 2 需要先完成上方的 Secrets 配置。

## 本地预检

推送前在本地跑一遍和 CI Job 1 等价的检查：

```bash
make ci-check
```

## 本地构建与推送镜像

```bash
# 构建
make docker-build DOCKER_USERNAME=你的用户名

# 推送（需要先 docker login）
make docker-push DOCKER_USERNAME=你的用户名
```
