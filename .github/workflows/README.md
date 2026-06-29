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

使用腾讯云容器镜像服务 TCR（个人版免费，每月 500 个镜像免费存储）。

### 前提：开通 TCR 个人版

1. 打开 [腾讯云容器镜像服务控制台](https://console.cloud.tencent.com/tcr)
2. 首次使用会提示开通，选择**个人版**（免费额度），确认开通
3. 开通后进入实例列表，点击 **ccr.ccs.tencentyun.com**（个人版默认实例）

### 第一步：创建命名空间

命名空间是镜像的组织目录，最终镜像路径为：

```
ccr.ccs.tencentyun.com/<命名空间>/xys-user:latest
```

操作步骤：
1. 在 TCR 控制台左侧点击 **命名空间**
2. 点击 **新建**，填写：
   - 命名空间名称：`xys-clone`
   - 访问级别：选择 **公开**（拉取不需要登录，推送仍需凭证）
3. 点击提交

> 名称也可以改成你喜欢的，比如 `open-the-bible`，但需要同步修改 `ci-cd.yaml` 和 `Makefile` 中的 `TCR_NAMESPACE`。

### 第二步：获取账号 ID（login 用户名）

1. 在腾讯云控制台右上角，点击你的头像
2. 点击 **账号信息**
3. 找到 **账号 ID**（一串数字，如 `100012345678`），复制下来

### 第三步：创建长期访问凭证（login 密码）

1. 回到 TCR 控制台，左侧点击 **访问凭证**
2. 点击 **新建**，会生成一个用户名和密码（token）
   - 用户名默认与你的账号 ID 一致
   - 密码是一长串字符，**立即复制保存**，关闭后不可查看
3. 如果已经有凭证，也可以直接点击 **查看密码** 来获取

### 第四步：填入 GitHub Secrets

打开仓库 → **Settings → Secrets and variables → Actions → New repository secret**，添加：

| Secret | 填什么 | 来源 |
|--------|--------|------|
| `DOCKER_USERNAME` | 你的账号 ID（如 `100012345678`） | 第二步获取 |
| `DOCKER_PASSWORD` | 长期访问凭证的密码 | 第三步获取 |

### 对应关系

```
┌──────────────────────────────────────────────────────────┐
│                        GitHub Secrets                     │
│  DOCKER_USERNAME = 100012345678 ──▶ docker login 用户名  │
│  DOCKER_PASSWORD = xxxxxxxx     ──▶ docker login 密码    │
└──────────────────────────────────────────────────────────┘
                           │
                           ▼
                   docker login ccr.ccs.tencentyun.com
                           │
                           ▼
┌──────────────────────────────────────────────────────────┐
│                   ci-cd.yaml 环境变量                      │
│  REGISTRY      = ccr.ccs.tencentyun.com                   │
│  TCR_NAMESPACE = xys-clone                                │
└──────────────────────────────────────────────────────────┘
                           │
                           ▼
          docker push ccr.ccs.tencentyun.com/xys-clone/xys-user:latest
```

> `DOCKER_USERNAME`（登录用的账号 UIN）和 `TCR_NAMESPACE`（命名空间名）是不同概念。
> - 本地 build：`make docker-build DOCKER_USERNAME=你的UIN`
> - CI build：workflow 自动读取 Secrets + env 组合

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
