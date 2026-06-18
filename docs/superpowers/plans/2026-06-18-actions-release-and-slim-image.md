# GitHub Actions 发布与镜像瘦身实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将本地构建/发布流程迁移到 GitHub Actions，并把 Docker 运行镜像改为更小的运行时镜像。

**架构：** CI、分支 Docker 镜像、正式 Release 拆成独立 workflow。正式 Release 先生成可复用的静态资源和 Linux 二进制，再复用这些产物构建多架构镜像。Docker 运行阶段只安装运行依赖并只复制运行脚本。

**技术栈：** GitHub Actions、Docker Buildx、Node 22、Go 1.25、Alpine 3.20、shell scripts、GitHub/Gitee Release API。

---

## 文件结构

- 修改：`.github/workflows/beta.yml`，改为调用统一瘦身后的 Docker 构建路径，保留 `dev` 分支 `beta` 镜像。
- 修改：`.github/workflows/feature.yml`，更新 action 版本、输出写法和多架构构建。
- 创建：`.github/workflows/ci.yml`，负责前端生产构建和后端可执行文件构建。
- 创建：`.github/workflows/release.yml`，负责 tag 发布、归档包、Docker 多架构镜像、GitHub/Gitee Release 和通知。
- 创建：`.github/scripts/package-release-asset.sh`，把单个平台二进制、`web_statics`、运行脚本和图标打成 zip/tar.gz。
- 创建：`.github/scripts/package-fnos.sh`，在 Actions 中准备 FNOS app 目录并在存在 `fnpack` 时生成 `.fpk`。
- 创建：`.github/scripts/publish-gitee-release.sh`，在有 `GITEE_ACCESS_TOKEN` 时创建 Gitee Release 并上传附件。
- 创建：`.github/scripts/notify-release.sh`，在有通知 secrets 时发送 Telegram/MeoW 通知。
- 修改：`Dockerfile`，源代码多阶段构建改用官方 Node/Go builder 和 Alpine runtime。
- 修改：`Dockerfile_base`，表达新的 runtime base 内容，作为可选基底镜像来源。
- 修改：`Dockerfile_beta`，改用 Alpine runtime 并只复制运行必需文件。
- 修改：`Dockerfile_local`，改用 Alpine runtime，继续支持 `TARGETARCH` 二进制选择。
- 创建：`Dockerfile_beta.dockerignore`，为 beta 产物镜像保留根二进制和静态资源上下文。
- 创建：`Dockerfile_local.dockerignore`，为本地/release 产物镜像保留 `temp_build` 和静态资源上下文。
- 修改：`backend/scripts/docker-entrypoint.sh`，去掉对 `getent` 的硬依赖，保持瘦身 Alpine 可运行。
- 修改：`.dockerignore`，允许 Dockerfile 需要的 workflow 外产物进入构建上下文，同时排除无关源码和构建产物。
- 修改：`.gitignore`，补充 GitHub Actions 下载/展开产物和本地 Docker 临时产物忽略项。

## 任务 1：新增 CI 与发布辅助脚本

**文件：**
- 创建：`.github/scripts/package-release-asset.sh`
- 创建：`.github/scripts/package-fnos.sh`
- 创建：`.github/scripts/publish-gitee-release.sh`
- 创建：`.github/scripts/notify-release.sh`
- 创建：`.github/workflows/ci.yml`

- [x] **步骤 1：创建脚本文件**

脚本必须使用 `set -euo pipefail`，入口参数固定，不能依赖交互输入。`package-release-asset.sh` 接收 `tag os arch binary web_statics out_dir`，生成 `QMediaSync_<os>_<arch>.zip` 或 `.tar.gz`。

- [x] **步骤 2：创建 CI workflow**

`ci.yml` 在 PR、`main`、`dev`、`feature/**` 推送时运行。前端执行 `npm ci` 和 `npm run build`；后端执行 `go mod download` 和 `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tmp/QMediaSync .`。

- [x] **步骤 3：本地校验脚本语法**

运行：`bash -n .github/scripts/package-release-asset.sh .github/scripts/package-fnos.sh .github/scripts/publish-gitee-release.sh .github/scripts/notify-release.sh`

预期：无输出，退出码 0。

## 任务 2：瘦身 Docker 运行镜像

**文件：**
- 修改：`Dockerfile`
- 修改：`Dockerfile_base`
- 修改：`Dockerfile_beta`
- 修改：`Dockerfile_local`
- 修改：`backend/scripts/docker-entrypoint.sh`
- 修改：`.dockerignore`
- 修改：`.gitignore`

- [x] **步骤 1：替换运行阶段依赖**

运行阶段统一基于 `alpine:3.20`，安装 `ca-certificates tzdata ffmpeg inotify-tools postgresql15`，不安装 `curl bash postgresql15-client`，不写 `/etc/sudoers`。

- [x] **步骤 2：缩小复制范围**

Docker 镜像只复制 `/app/QMediaSync`、`/app/web_statics`、`/app/scripts/docker-entrypoint.sh`、`/app/scripts/watch_update.sh`、`/app/icon.ico`。

- [x] **步骤 3：更新入口脚本兼容性**

`docker-entrypoint.sh` 新增 group 检查函数，优先使用 `getent`，不存在时读取 `/etc/group`。这样 Alpine runtime 不需要额外安装 `getent` 来源包。

- [x] **步骤 4：本地构建验证**

运行：`docker build -t qmediasync:ci .`

预期：镜像构建成功，运行阶段不再从 `qicfan/qms-build-base` 拉取。

## 任务 3：迁移分支 Docker 镜像到新版 Actions

**文件：**
- 修改：`.github/workflows/beta.yml`
- 修改：`.github/workflows/feature.yml`

- [x] **步骤 1：替换过期输出写法**

所有 `::set-output` 改为 `$GITHUB_OUTPUT`。

- [x] **步骤 2：升级 action 版本**

使用 `actions/checkout@v4`、`docker/setup-qemu-action@v3`、`docker/setup-buildx-action@v3`、`docker/login-action@v3`、`docker/build-push-action@v6`。

- [x] **步骤 3：启用多架构和缓存**

分支镜像使用 `Dockerfile` 源码构建，平台为 `linux/amd64,linux/arm64`，使用 `cache-from: type=gha` 和 `cache-to: type=gha,mode=max`。

## 任务 4：新增正式 Release workflow

**文件：**
- 创建：`.github/workflows/release.yml`

- [x] **步骤 1：构建前端静态资源**

tag `v*` 和手动触发时运行，使用 Node 22，执行 `npm ci` 和 `npm run build`，上传 `backend/web_statics` 为 artifact。

- [x] **步骤 2：交叉编译并打包归档**

矩阵包含 `windows/amd64`、`windows/arm64`、`linux/amd64`、`linux/arm64`。每个平台注入版本、发布日期和 API secrets，运行 `go build -trimpath`，调用 `.github/scripts/package-release-asset.sh` 生成 release assets。

- [x] **步骤 3：Docker 多架构 release 镜像**

下载 Linux 二进制和 `web_statics`，放入 `temp_build/` 和 `backend/web_statics/`，使用 `Dockerfile_local` 构建并推送 `tag` 和 `latest`。

- [x] **步骤 4：FPK 可选打包**

下载 Linux 二进制和 `web_statics`，如果 `FNPACK_DOWNLOAD_URL` 配置则下载安装；如果存在 `fnpack` 则生成 `QMediaSync_amd64.fpk` 和 `QMediaSync_arm64.fpk`，否则跳过。

- [x] **步骤 5：创建 Release、Gitee 和通知**

读取 `.changes/<tag>.md` 作为 release notes，不存在时使用 `Release <tag>`。上传 zip、tar.gz 和存在的 fpk。Gitee 与通知 secrets 未配置时跳过。

## 任务 5：验证并整理

**文件：**
- 检查：所有修改文件

- [x] **步骤 1：workflow 结构检查**

运行：`git diff --check`

预期：无空白错误。

- [x] **步骤 2：构建命令验证**

运行：`(cd frontend && npm ci && npm run build)`，然后运行 `(cd backend && GOSUMDB=off CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tmp/QMediaSync .)`。

预期：两条命令退出码 0。

- [x] **步骤 3：Docker 验证**

运行：`docker build -t qmediasync:ci .`

预期：镜像构建成功。

- [x] **步骤 4：工作区检查**

运行：`git status --short`

预期：只有本次迁移修改和新增文件，没有构建产物进入待提交状态。
