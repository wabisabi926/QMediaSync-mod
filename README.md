# QMediaSync

### 开源版本不包含115开放平台账号，需要自备

### 本项目接受除了资源（搜索、订阅、下载）、逆向接口的一切功能PR

## 介绍

- **默认用户名 admin,密码 admin123**
- 默认端口：http-12333   https-12332
- emby代理端口默认：http-8095  https-8094

## 调试启动

后端：

```bash
cd backend
go run .
```

前端：

```bash
cd frontend
npm install
npm run dev
```

前端开发环境默认连接 `http://localhost:12333/api`。

## 退出

- linux: ```ctrl + c```
- windows: 系统托盘找到QMediaSync图标，右键退出

## 发布新版本

更新日志由 [git-cliff](https://git-cliff.org/) 从 git 提交记录自动生成，因此提交信息需遵循 [Conventional Commits](https://www.conventionalcommits.org/)（`feat:`、`fix:`、`docs:`、`chore:`、`ci:` 等前缀），不规范的提交（如 `Merge`、自由文案）会被自动忽略。

发版步骤（本地执行）：

1. 生成本版本 changelog（需先安装 git-cliff，见其[安装文档](https://git-cliff.org/docs/installation/)）：

   ```bash
   scripts/release/gen-changelog.sh v0.xx.xx
   ```

   该脚本会：
   - 读取上一个 `v*` 标签至今的提交，按类型分组生成 `.changes/v0.xx.xx.md`（作为 GitHub Release 正文）；
   - 把本版本段落插入 `CHANGELOG.md` 顶部，保留历史内容。

2. 检查生成内容无误后提交：

   ```bash
   git add CHANGELOG.md .changes
   git commit -m "chore: release v0.xx.xx"
   ```

3. 打标签并推送，触发发布：

   ```bash
   git tag v0.xx.xx
   git push origin v0.xx.xx
   ```

推送 `v*` 标签会触发 GitHub Actions 的 release 流程，生成 Windows/Linux 发布包、可选的飞牛 FPK，并创建 GitHub Release。
GitHub Release 的正文取自上一步提交的 `.changes/v0.xx.xx.md`；若该文件不存在，正文回退为占位文字 `Release <tag>`。
发布流程还会使用 `GITHUB_TOKEN` 推送 GHCR 镜像 `ghcr.io/<owner>/qmediasync:<tag>` 和 `ghcr.io/<owner>/qmediasync:latest`。

也可以在 GitHub Actions 中手动触发 `release` workflow，并输入要发布的 Git tag（同样要求该 tag 对应的 `.changes/<tag>.md` 已提交）。

> 飞牛 FPK 打包依赖飞牛官方工具 `fnpack`（不公开分发）。release workflow 通过仓库 Secret `FNPACK_DOWNLOAD_URL`（指向可下载 `fnpack` 可执行文件的地址）下载安装，再用 `backend/FNOS/` 下的素材执行 `fnpack build` 生成 `*.fpk`。**未配置该 Secret 时，`fpk` job 和 `scripts/release/package-fnos.sh` 会自动跳过 FPK 打包**，其余 Windows/Linux 发布包、Docker 镜像不受影响；若希望缺少工具时直接报错（而非静默跳过），可在脚本环境设 `REQUIRE_FNPACK=1`。

> 调整 changelog 的分组、过滤规则可编辑仓库根目录的 `cliff.toml`。

## 数据库

开源版本不包含postgres数据库二进制文件，需要自己安装，建议版本15.x，然后配置环境变量使用。

## 需要自备的密钥

- 115开放平台 AppID，现在改为使用OAuth授权方式，开发者需要根据代码自己实现OAUTH服务端来和115通信，或者改为二维码扫码登录授权。
- TMDB API KEY，可在 web 页面（刮削设置）填写；刮削实际使用 v3 API Key
- OpenAI兼容的 API KEY，目前用的硅基流动，可在 web 页面（刮削设置）填写
- Fanart.tv API KEY，可在 web 页面（刮削设置）填写

以上 key 可在 `backend/main.go` 开头的变量中设置、编译时通过 ldflags 传入，或运行时通过环境变量 / `config/.env` 注入（变量名 `TMDB_API_KEY`、`TMDB_ACCESS_TOKEN`、`SC_API_KEY`、`FANART_API_KEY`，无 `DEFAULT_` 前缀）。取值优先级：web UI > 环境变量 > ldflags。

> 网盘 OAuth 加密密钥 `ENCRYPTION_KEY` 无需自备：每个实例首次启动自动生成并保存到 `config/encryption.key`，也可用环境变量 `ENCRYPTION_KEY` 覆盖。

## 仓库结构

```text
backend/          Go 后端、内置静态前端产物
docker/           Dockerfile、容器入口脚本和在线更新监视脚本
frontend/         Vue/Vite 前端源码
scripts/release/  GitHub Actions 发布打包辅助脚本、changelog 生成脚本
scripts/install/  Linux 裸机安装辅助脚本
.github/          CI 构建流程
cliff.toml        git-cliff 配置（从提交记录生成 changelog）
```

前端生产构建会输出到 `backend/web_statics`，后端从该目录提供 Web UI。

## 原项目地址

本仓库基于以下原项目合并而来：

- 后端：[qicfan/qmediasync](https://github.com/qicfan/qmediasync)
- 前端：[qicfan/q115-strm-frontend](https://github.com/qicfan/q115-strm-frontend)
- Wiki：[qicfan/qmediasync/wiki](https://github.com/qicfan/qmediasync/wiki)
