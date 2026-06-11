# D2api 构建&部署流程

> 最后更新：2026-06-11

---

## 仓库地址

| Remote | URL |
|--------|-----|
| origin (fork) | `https://github.com/Acidmoon/D2api` |
| upstream (原项目) | `https://github.com/Wei-Shaw/sub2api` |
| dev 分支 | `origin/dev` |

---

## 镜像类型

| 镜像 | tag | 触发方式 | 用途 |
|------|-----|---------|------|
| dev 测试镜像 | `ghcr.io/acidmoon/d2api:dev` | push `dev` 分支 | 服务器 8081 端口测试 |
| release 生产镜像 | `ghcr.io/acidmoon/d2api:latest` + `:版本号` | push `v*-d2api` tag | 服务器 8080 端口生产 |

---

## 构建流程

### Dev 构建 (`build-dev` job)

1. `pnpm install --frozen-lockfile` → 安装前端依赖
2. `npx vite build` → 编译前端（`frontend/dist/` → `backend/internal/web/dist/`
   - 用 `npx vite build` 而不是 `pnpm run build`，跳过 `vue-tsc -b` 类型检查
   - 类型检查由 CI 的 `make test-frontend` 单独负责，不需要在 Docker 构建中重复
3. `go build -tags embed` → 编译后端（嵌入前端 dist）
4. `docker build -f Dockerfile` → 多阶段构建打包为 Alpine 镜像（4 个 Stage）
5. 推送到 `ghcr.io/acidmoon/d2api:dev`

### Release 构建 (`build-release` job)

1. 前端：`pnpm install` → `pnpm run build`（含 vue-tsc 类型检查）
2. 后端：GoReleaser 用 `.goreleaser.d2api.yaml` 编译 → `Dockerfile.goreleaser` 打包
3. 推送到 `ghcr.io/acidmoon/d2api:latest` + `:0.1.x`

---

## 避坑记录

### 1. Dockerfile 中前端构建必须跳过 vue-tsc
`package.json` 的 `build` 脚本是 `vue-tsc -b && vite build`，但 Docker 容器内 `vue-tsc -b` 会在类型检查阶段失败。
→ 把 `RUN pnpm run build` 改成 `RUN npx vite build`（Dockerfile 第 32 行）。

### 2. Dockerfile.goreleaser 的 COPY 文件名
`.goreleaser.d2api.yaml` 编译的二进制叫 `d2api`，但 `Dockerfile.goreleaser` 里 `COPY sub2api`。
→ 改为 `COPY d2api /app/sub2api`（Dockerfile.goreleaser 第 46 行，目标路径 `/app/sub2api` 不动）。

### 3. Node 镜像版本
原项目 Dockerfile 默认 `node:24-alpine`，但 `pnpm run build` 在 Node 24 下失败。
→ 降为 `node:20-alpine`（与 CI release job 一致）。

### 4. Go embed 需要前端 dist 路径正确
`vite.config.ts` 配置了 `outDir: '../backend/internal/web/dist'`，所以 Docker 里 `npx vite build` 输出自动落到正确路径。如果在 CI 分步构建，需要手动 `cp -r frontend/dist backend/internal/web/dist`。

### 5. 发行版 tag 格式
必须是 `v*-d2api`（如 `v0.1.0-d2api`），才会触发 `build-release` job。

---

## 关键文件

| 文件 | 作用 |
|------|------|
| `Dockerfile` | dev 构建用，多阶段：前端→后端→运行时 |
| `Dockerfile.goreleaser` | release 构建用，只打包预编译的二进制 |
| `.goreleaser.d2api.yaml` | release 用 GoReleaser 配置，镜像 `ghcr.io/acidmoon/d2api` |
| `.github/workflows/build-d2api.yml` | CI 工作流，`dev` 和 `release` 两个 job |
| `frontend/vite.config.ts:64` | `outDir: '../backend/internal/web/dist'` — 前端编译输出路径 |
| `frontend/Dockerfile`（根目录） | 主 Dockerfile，SKIP vue-tsc，node:20 |

---

## 服务器部署

### Dev 实例（8081 测试）

```bash
docker pull ghcr.io/acidmoon/d2api:dev
docker compose -f docker-compose.dev.yml up -d
```

### 生产实例（8080 上线）

**首次切换**（一次性）：
```bash
# 修改 docker-compose.yml
# image: weishaw/sub2api:latest  →  image: ghcr.io/acidmoon/d2api:latest
docker compose pull && docker compose up -d
```

**日常更新**：
```bash
docker compose pull && docker compose up -d
```

### 回滚到原项目

```bash
sed -i 's|ghcr.io/acidmoon/d2api:latest|weishaw/sub2api:latest|' docker-compose.yml
docker compose pull && docker compose up -d
```

当前版本迁移文件一致，数据库无需恢复。

---

## 发布清单

- [ ] 代码推送到 `dev` 分支 → 等待 CI 构建 `:dev` → 8081 测试验证
- [ ] 确认无误后打 tag：`git tag v0.1.x-d2api && git push origin --tags`
- [ ] 等待 CI 构建 `:latest` → 服务器 `docker compose pull && docker compose up -d`
