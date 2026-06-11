# D2api 构建&部署流程

> 最后更新：2026-06-11（修正 dist 输出路径说明）

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
2. `npx vite build` → 编译前端
   - 用 `npx vite build` 而不是 `pnpm run build`，跳过 `vue-tsc -b` 类型检查
   - 类型检查由 CI 的 `make test-frontend` 单独负责，不需要在 Docker 构建中重复
   - **产物直接输出到 `backend/internal/web/dist/`**（由 `vite.config.ts` 的 `outDir: '../backend/internal/web/dist'` 决定），**无需任何 cp 拷贝**
3. `test -f backend/internal/web/dist/index.html` → 校验前端产物已就位（供 go embed）
4. `go build -tags embed` → 编译后端（嵌入前端 dist）
5. `docker build -f Dockerfile.goreleaser` → 打包为镜像
6. 推送到 `ghcr.io/acidmoon/d2api:dev`

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

### 4. Go embed 的前端 dist 路径（重要）
`vite.config.ts` 配置了 `outDir: '../backend/internal/web/dist'`，所以**无论本地还是 CI**，`vite build` 都直接把产物输出到 `backend/internal/web/dist/`，go embed 能直接读到。

⚠️ **不要画蛇添足加 `cp -r frontend/dist backend/internal/web/dist`**：`frontend/dist` 根本不存在（产物没输出到那），这个 cp 会 `exit 1` 让 `build-dev` 失败。曾在 commit `7d041b46` 误加、`fef9bec2` 修复。CI 里改用 `test -f backend/internal/web/dist/index.html` 校验产物存在即可。

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

### 推送前本地预检（前端有改动时务必执行）

CI 的 `build-dev` 用 `npx vite build`（比 dev server 严格，PostCSS/构建错误会直接失败），所以前端改动后**先在本地复现 CI 同款命令**，避免 push 后才被 CI 卡住：

```bash
cd frontend
npx vite build                          # 等价 CI build-dev，须 exit 0
test -f ../backend/internal/web/dist/index.html && echo "dist OK"   # 校验产物
node node_modules/vue-tsc/bin/vue-tsc.js --noEmit   # 类型检查(对应 make test-frontend)
```

> 注：`backend/internal/web/dist/` 是 `vite build` 的输出目录（由 outDir 决定），**不要手动 cp**（见避坑 #4）。

### 发布步骤

- [ ] 前端改动已通过上面的本地预检
- [ ] 代码推送到 `dev` 分支 → 等待 CI 构建 `:dev` → 8081 测试验证
- [ ] 确认无误后打 tag：`git tag v0.1.x-d2api && git push origin --tags`
- [ ] 等待 CI 构建 `:latest` → 服务器 `docker compose pull && docker compose up -d`
