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
- 网盘文件浏览器使用 `/api/path/files` 拉取当前目录视图。前端传入 `account_id`、`path`、`page`、`page_size` 和 `refresh`；`page_size` 是 UI 每页数量，支持 `50`、`100`、`200`、`500`，默认 `50`。后端会按网盘来源能力批量请求上游并缓存批次，普通翻页优先从缓存切片返回。
- `/api/path/files` 响应 `data` 保持 `{ list, total, page, page_size }` 兼容字段，并新增 `total_exact`、`has_more`、`sort_by`、`sort_order` 和 `cache`。115 和 OpenList 的 `total_exact=true`；百度网盘没有精确总数，`total_exact=false`，`total` 只用于让分页继续前进。
- 手动刷新传 `refresh=1`，后端清当前目录、当前排序视图和筛选条件对应的缓存并重新请求上游；创建目录、删除文件或目录成功后会失效同一目录下所有排序视图缓存。OpenList 普通列表请求使用 `refresh=false`，只有手动刷新或写操作后的重新加载才让第一轮上游请求使用 `refresh=true`。
- 文件管理器排序入口暂时整体隐藏，前端不提交 `sort_by` 和 `sort_order`。后端接口仍保留这两个查询参数作为兼容能力，但当前只使用上游支持的全局排序参数，不做当前页或单个缓存批次的本地排序。已知 115 Open API 在部分目录中返回顺序与请求排序参数不一致；OpenList 第一版也只使用默认顺序。后续应在后端实现完整目录排序视图缓存后，再恢复前端排序入口。
- 上传 / 下载队列列表响应包含 `queue_status` 快照，前端批量按钮状态以该快照为准；暂停 / 恢复只依据队列运行态，清理 / 重试类操作再依据任务数量；WebSocket 状态事件只更新运行标记，最终仍以 HTTP 快照校准。
- 上传队列列表会在 115 任务上补齐 `upload_phase`、`progress_percent`、`uploaded_bytes`、`total_parts`、`uploaded_parts`、`upload_result` 和 `resume_state` 等面向展示的字段。`upload_phase` 和 `upload_result` 保持后端机器值，前端在上传队列“阶段 / 结果”列映射为用户可理解的文案，例如 `pending` 显示为“等待上传”、`checking_remote` 显示为“检查远端文件”、`remote_completed_pending_finalize` 显示为“等待收尾”、`skipped_after_rapid_wait` 显示为“秒传等待超时，已跳过上传”。`upload_queue_changed` 事件在进度变化时也会带同名 patch 字段，进度事件按任务节流到约 1 秒；创建、状态切换、完成和失败事件不按进度节流。
- 上传完成、远端已存在跳过和后续 STRM Webhook 使用 `strm_generation_tasks` 作为统一 STRM 生成队列。后端启动时会把遗留的 `running` STRM 生成任务恢复为 `pending`，后台 worker 轮询处理；worker 停止时由 context 取消中断的任务保持 `running` 或 `finalizing`，不计为失败，下一次启动再恢复或继续收尾，且取消后不再提交 Emby 刷新或执行源文件清理。文件级任务生成或确认 STRM 后只更新 `SyncFile` 和 STRM / 元数据本地文件，不创建 `syncs` 同步记录。非 Webhook 文件任务在 STRM 发生变化时提交 Emby 刷新；Webhook 文件任务只有 `refresh_emby=true` 且 STRM 变更或新增元数据下载任务时才提交刷新，批量和目录扫描只在父任务所有子任务成功完成后统一提交。成功生成的子任务会先累计父任务进度并进入 `finalizing`，父任务刷新提交成功或父任务尚未满足提交条件后才标记 `completed`；`finalizing` 属于活跃状态，会参与 `request_hash` 去重并被 worker 继续扫描，避免刷新失败后子任务不可恢复或重试重复累计父进度。父任务只有 `status=completed`、计数满足且存在 STRM / 元数据变化时才提交批量刷新，`failed` 和 `waiting_children` 不提交。目录监控上传的元数据文件会在源文件清理前复制到 STRM 本地路径；复制失败时任务失败且不会触发源文件清理。单文件后处理只执行一次 STRM 内容比较；确认需要更新后直接写入文件，不重复输出差异 WARN，也不重复输出完整同步启动时的“生效 STRM 配置”INFO。
- WebSocket 事件默认只用于通知状态或列表可能发生变化，前端收到结构性事件后按当前页、筛选条件重新拉取快照；上传队列的 `upload_queue_changed` 进度 patch 例外，当前页已有任务会优先局部合并 `uploaded_bytes`、`progress_percent`、`upload_speed_bytes`、`upload_phase`、`upload_result`、`resume_state` 和分片字段，创建、清理、重试等结构性变化仍重新拉取快照。
- 长任务进度优先使用事件推送；保留 HTTP 状态接口作为首次加载、刷新恢复和 WebSocket 断线兜底。
- 115 OAuth 和二维码授权属于短生命周期外部授权流程，继续使用现有轮询，不接入通用队列事件。
- 新增轮询必须具备页面隐藏暂停、请求防重叠和卸载清理。

### 前端路由和浏览器历史

- 页面标题只在 Vue Router 成功完成导航后更新，避免目标页标题写入上一条浏览器历史记录。
- 登录成功、退出登录、登录失效和鉴权守卫跳转使用 `router.replace` 或带 `replace: true` 的重定向，避免把登录页、失效页或被拦截页面留在历史栈里。
- 详情页和表单页的“返回 / 取消”使用 `navigateBackOrReplace(router, fallback)`：存在 Vue Router 记录的应用内上一页时执行浏览器回退，直接打开深链或没有上一页时替换到兜底列表页。
- 表单保存成功后直接 `replace` 到列表页，不复用“返回 / 取消”逻辑，避免保存后的表单页继续停留在浏览器后退栈顶部。

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
- 分页列表统一使用 `frontend/src/components/common/ResponsivePagination.vue`；页面保留页码、每页数量和数据加载逻辑，分页组件只负责响应式布局和事件透传。移动端默认保留每页数量选择。
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
