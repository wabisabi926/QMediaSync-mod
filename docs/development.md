# 开发调试

## 后端启动

```bash
cd backend
go run .
```

如果没有 `config/config.yaml`，后端会先启动配置向导，默认监听 `http://127.0.0.1:12333`。配置保存后需要重新启动后端。

## 前端启动

前端依赖要求 Node `>=22.22.2`，pnpm 使用 11.x。

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
- 网盘文件浏览器使用 `/api/path/files` 拉取当前页，响应 `data` 为 `{ list, total, page, page_size }`；可获取目录总数的网盘应把服务端总数写入 `total`，前端分页页码不能用当前页 `list.length` 推断。
- 上传 / 下载队列列表响应包含 `queue_status` 快照，前端批量按钮状态以该快照为准；暂停 / 恢复只依据队列运行态，清理 / 重试类操作再依据任务数量；WebSocket 状态事件只更新运行标记，最终仍以 HTTP 快照校准。
- WebSocket 事件只用于通知状态或列表可能发生变化，前端收到事件后按当前页、筛选条件重新拉取快照，不在客户端维护分页列表增量状态。
- 长任务进度优先使用事件推送；保留 HTTP 状态接口作为首次加载、刷新恢复和 WebSocket 断线兜底。
- 115 OAuth 和二维码授权属于短生命周期外部授权流程，继续使用现有轮询，不接入通用队列事件。
- 新增轮询必须具备页面隐藏暂停、请求防重叠和卸载清理。

### 通知渠道热刷新

- 渠道创建、更新、启用、禁用和删除时，通知管理器同步更新内存里的 handler 与规则，保证接口返回后发送路由已使用最新配置。
- 需要后台监听的渠道不能在 HTTP 请求路径中执行外部网络初始化；例如 Telegram Bot 的初始化、菜单设置和长轮询监听必须由 handler 自身在后台协程中完成。
- 规则启用状态变化只刷新规则映射，不重建渠道 handler，避免影响正在运行的后台监听。

### 前端反馈和按钮约定

- 设置页保存类主操作优先使用绿色按钮（`type="success"`），按钮文案和成功反馈保持同一动作语义。
- 启动、恢复运行等正向流程动作使用 `type="success"`；暂停、停止、重试、恢复备份和数据库修复等有风险但非删除动作使用 `type="warning"`。
- 删除、清空、撤销等不可恢复或破坏性动作使用 `type="danger"`，并保留确认步骤。
- 测试连接、搜索、生成、添加等中性主动作使用 `type="primary"`；刷新、复制、下载、取消和清除选择等辅助动作使用默认或信息类样式。
- 设置页底部主操作使用 `size="large"`；工具栏使用默认尺寸；表格行内操作使用 `size="small"`、`link` 或 `text`。
- 普通动作按钮的图标使用 Element Plus 的 `:icon="IconName"` 属性，不在按钮内容里手写 `<el-icon>`；混用两种写法会导致图标和文字之间的内边距不一致。
- 下拉菜单按钮的右侧箭头、后缀状态图标等不属于主动作图标的 suffix affordance 可以保留手写 `<el-icon class="el-icon--right">`。
- 图标按钮应保持“图标 + 文字”作为整体居中，按钮内图标间距交给 Element Plus 处理，不在组件样式里为按钮内 `.el-icon` 额外设置 `margin-left` 或 `margin-right`。
- 保存页如果已经使用页面内状态提示（例如底部 `el-alert`）展示保存成功，不再额外弹出成功 toast，避免同一结果重复提示。
- 错误、校验失败、复制、测试连接、启动任务等需要即时注意或短生命周期反馈的动作，可以继续使用 `ElMessage`。

## 常用验证

时间字段策略见 [数据库 - 时间字段策略](database.md#时间字段策略)，Cron 边界见 [请求校验约定 - Cron 表达式边界](validation.md#cron-表达式边界)，Emby 同步架构见 [Emby 同步维护说明](emby-sync.md)。

```bash
(cd backend && go test ./...)
(cd frontend && pnpm lint)
(cd frontend && pnpm format:check)
(cd frontend && pnpm run type-check)
(cd frontend && pnpm run build)
```

后端生产构建默认关闭 Gin 的 MsgPack 绑定和渲染支持，以缩小二进制体积：

```bash
(cd backend && CGO_ENABLED=0 go build -trimpath -tags=nomsgpack -ldflags="-s -w" -o QMediaSync .)
```

前端生产构建输出到 `frontend/dist`。发布、Docker 和离线包会把该目录作为 Web UI 静态资源输入，并在最终运行目录中放置为 `web_statics`。

## 退出

- Linux：按 `Ctrl+C` 退出。
- Windows：在系统托盘找到 QMediaSync 图标，右键退出。
