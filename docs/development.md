# 开发调试

## 后端启动

```bash
cd backend
go run .
```

如果没有 `config/config.yaml`，后端会先启动配置向导，默认监听 `http://127.0.0.1:12333`。配置保存后需要重新启动后端。

## 前端启动

```bash
cd frontend
corepack enable
corepack prepare pnpm@11 --activate
pnpm install
pnpm run dev
```

前端开发环境默认连接 `http://localhost:12333/api`。

后端默认允许 Vite 开发来源 `http://localhost:5173`、`http://127.0.0.1:5173` 和 `http://[::1]:5173` 携带浏览器登录 Cookie。若修改 Vite 端口、使用自定义开发域名，或让前端跨源访问后端，需要把前端来源加入 `config/config.yaml`：

```yaml
trustedOrigins:
  - http://dev.example.local:5173
```

### 前端状态刷新和 WebSocket 事件

- 分页列表接口保留 HTTP 快照语义，例如上传 / 下载队列列表仍通过 `/api/upload/queue` 和 `/api/download/queue` 拉取当前页。
- WebSocket 事件只用于通知状态或列表可能发生变化，前端收到事件后按当前页、筛选条件重新拉取快照，不在客户端维护分页列表增量状态。
- 长任务进度优先使用事件推送；保留 HTTP 状态接口作为首次加载、刷新恢复和 WebSocket 断线兜底。
- 115 OAuth 和二维码授权属于短生命周期外部授权流程，继续使用现有轮询，不接入通用队列事件。
- 新增轮询必须具备页面隐藏暂停、请求防重叠和卸载清理。

## 常用验证

```bash
(cd backend && go test ./...)
(cd frontend && pnpm lint)
(cd frontend && pnpm format:check)
(cd frontend && pnpm run type-check)
(cd frontend && pnpm run build)
```

前端生产构建输出到 `frontend/dist`。发布、Docker 和离线包会把该目录作为 Web UI 静态资源输入，并在最终运行目录中放置为 `web_statics`。

## 退出

- Linux：按 `Ctrl+C` 退出。
- Windows：在系统托盘找到 QMediaSync 图标，右键退出。
