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

```bash
git tag vx.xx.xx
git push origin vx.xx.xx
```

推送 `v*` 标签会触发 GitHub Actions 的 release 流程，生成 Windows/Linux 发布包、可选的飞牛 FPK，并创建 GitHub Release。
发布流程会使用 `GITHUB_TOKEN` 推送 GHCR 镜像 `ghcr.io/<owner>/qmediasync:<tag>` 和 `ghcr.io/<owner>/qmediasync:latest`。

也可以在 GitHub Actions 中手动触发 `release` workflow，并输入要发布的 Git tag。

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
scripts/release/  GitHub Actions 发布打包辅助脚本
scripts/install/  Linux 裸机安装辅助脚本
.github/          CI 构建流程
```

前端生产构建会输出到 `backend/web_statics`，后端从该目录提供 Web UI。

## 原项目地址

本仓库基于以下原项目合并而来：

- 后端：[qicfan/qmediasync](https://github.com/qicfan/qmediasync)
- 前端：[qicfan/q115-strm-frontend](https://github.com/qicfan/q115-strm-frontend)
- Wiki：[qicfan/qmediasync/wiki](https://github.com/qicfan/qmediasync/wiki)
