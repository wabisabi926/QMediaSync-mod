# GitHub Actions 发布与镜像瘦身设计

## 背景

当前本地发布流程集中在 `scripts/release/build_and_release.sh`，同时负责前端构建、Go 交叉编译、归档打包、FPK 打包、Docker 多架构镜像、GitHub/Gitee Release 和通知。这个脚本依赖本地环境、交互式清理、Docker 登录状态和 `fnpack`，不适合作为长期发布入口。

Docker 运行镜像当前基于 `qicfan/qms-build-base`，镜像内包含构建与运行混合依赖，并复制整个 `backend/scripts`。运行容器实际需要的是 QMediaSync 可执行文件、前端静态资源、Docker 入口脚本、更新监视脚本、PostgreSQL 嵌入式运行依赖、ffmpeg、时区和 CA 证书。

## 目标

将本地构建和发布入口迁移到 GitHub Actions，同时缩小运行镜像并减少重复构建成本。

## 方案

采用 C 方案：Actions 原生发布流程加运行镜像瘦身。

1. CI 负责验证前端生产构建和后端可执行文件构建，不把当前已知失败的 `go test ./...` 作为阻塞项。
2. Docker 分支镜像使用源代码多阶段构建，支持 `dev` 分支 `beta` 标签和 `feature/**` 分支标签。
3. 正式发布由 tag 触发，先构建一次前端静态资源，再按矩阵交叉编译 Windows/Linux amd64/arm64，并生成 zip/tar.gz 发布包。
4. 正式发布 Docker 镜像复用已构建的 Linux 二进制和静态资源，通过 `Dockerfile_local` 构建多架构镜像，避免 Docker 内重复构建前端和 Go。
5. FPK 打包迁移到 Actions 中的可选 job。仓库没有 `fnpack` 安装来源，因此 job 支持 `FNPACK_DOWNLOAD_URL`，未配置时跳过而不阻断 GitHub Release。
6. GitHub Release、Gitee Release 和通知迁移到 Actions。Gitee 与通知通过 secrets 控制，未配置时跳过。
7. 运行镜像改用 Alpine 运行时依赖，不再使用构建基底作为运行基底；只复制 Docker 运行需要的两个脚本。

## 镜像瘦身边界

保留：

- `postgresql15`：后端支持 embedded PostgreSQL，需要 `initdb`、`postgres`、`pg_ctl`。
- `ffmpeg`：Emby 302/视频预览相关运行能力需要。
- `inotify-tools`：`watch_update.sh` 依赖 `inotifywait`。
- `tzdata`、`ca-certificates`：时区和 HTTPS 访问需要。

移除：

- `curl`：运行容器入口脚本不依赖。
- `bash`：Docker 入口脚本使用 `/bin/sh`。
- `postgresql15-client`：应用不调用 `psql`。
- sudo 配置：容器入口以 root 启动并负责创建用户、改权限，不需要 sudo。
- `backend/scripts/linux-init.sh`：Linux 主机安装辅助脚本，不属于 Docker 运行时。

## 成功标准

- PR/分支 CI 能构建前端和后端。
- `dev` 和 `feature/**` 分支可推送 Docker 镜像。
- tag 发布能生成 Windows/Linux amd64/arm64 归档包并创建 GitHub Release。
- Docker release 镜像支持 linux/amd64 和 linux/arm64。
- 本地 `docker build` 使用瘦身后的运行镜像成功。
- 工作区不再依赖根目录 `QMediaSync` 或 `backend/web_statics` 被提交。
