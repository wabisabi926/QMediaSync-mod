# QMediaSync

![GitHub release (latest by date)](https://img.shields.io/github/v/release/qicfan/qmediasync)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/qicfan/qmediasync)

## 讨论方式

- 电报群：[http://t.me/q115_strm](https://t.me/q115_strm)
- QQ群：1057459156
- Meow官方频道：使用鸿蒙系统手机扫描下方二维码来关注频道（请用官方浏览器打开）
  
  <img src="https://s.mqfamily.top/meow.png" width="200" />

### 开源版本不包含115开放平台账号，需要自备

### 本项目接受除了资源（搜索、订阅、下载）、逆向接口的一切功能PR

#### PR以后如果没有动静可以邮件、TG、QQ联系作者

## 介绍

- **默认用户名 admin,密码 admin123**
- 默认端口：http-12333   https-12332
- emby代理端口默认：http-8095  https-8094
- 其他见 [wiki](https://github.com/qicfan/qmediasync/wiki)

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
如果配置了 `DOCKER_USERNAME` 和 `DOCKER_PASSWORD`，发布流程会同时推送 Docker Hub 镜像的版本标签和 `latest` 标签。

也可以在 GitHub Actions 中手动触发 `release` workflow，并输入要发布的 Git tag。

## 数据库

开源版本不包含postgres数据库二进制文件，需要自己安装，建议版本15.x，然后配置环境变量使用。详见wiki中的[安装](https://github.com/qicfan/qmediasync/wiki/Linux-%E5%AE%89%E8%A3%85%E4%BD%BF%E7%94%A8)

## 需要自备的密钥

- 115开放平台 AppID，现在改为使用OAuth授权方式，开发者需要根据代码自己实现OAUTH服务端来和115通信，或者改为二维码扫码登录授权。
- TMDB API KEY，可在web页面设置
- OpenAI兼容的 API KEY，目前用的硅基流动，可在web页面设置
- Fanart.tv API KEY

全部都在main.go文件中开头的变量中设置，也可以在编译时通过ldflags传入

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

## 贡献者

![Contributors](https://contrib.rocks/image?repo=qicfan/qmediasync)

## Star

![Star History](https://api.star-history.com/svg?repos=qicfan/qmediasync&type=Date)

## 请作者喝杯咖啡

![请作者喝杯咖啡](http://s.mqfamily.top/alipay_wechat.jpg)
