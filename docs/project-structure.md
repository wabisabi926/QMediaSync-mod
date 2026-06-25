# 项目结构

```text
backend/          Go 后端、嵌入的 Emby 302 代理、前端生产构建产物
docker/           Dockerfile、容器入口脚本和在线更新监视脚本
frontend/         Vue / Vite 前端源码
scripts/release/  GitHub Actions 发布打包辅助脚本、changelog 生成脚本和发布脚本共享函数
scripts/install/  Linux 裸机安装辅助脚本
.github/          CI/CD 工作流
.changes/         每个版本的 GitHub Release 正文
cliff.toml        git-cliff 配置（从提交记录生成 changelog）
```

前端生产构建会输出到 `backend/web_statics`，后端从该目录提供 Web UI；该目录是构建产物，不作为源码维护。

## 原项目地址

本仓库基于以下原项目合并而来：

- 后端：[qicfan/qmediasync](https://github.com/qicfan/qmediasync)
- 前端：[qicfan/q115-strm-frontend](https://github.com/qicfan/q115-strm-frontend)
- Wiki：[qicfan/qmediasync/wiki](https://github.com/qicfan/qmediasync/wiki)
