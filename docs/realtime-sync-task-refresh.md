# 同步任务实时刷新机制

同步任务详情使用 `/api/sync/tasks/:id/stream` 获取实时数据。该 stream 是详情页的唯一实时状态源，负责推送任务快照、状态 patch、运行日志、完成收尾和断线重同步信号。

## 总体流程

1. 后端创建、更新、完成、失败或删除同步记录后，发布同步任务结构化事件。
2. 全局事件 WebSocket 收到 `sync_task_created`、`sync_task_updated`、`sync_task_deleted`，用于同步记录页和同步目录页的局部刷新。
3. 同步任务详情打开 `/api/sync/tasks/:id/stream`，后端先订阅该任务事件，再重读数据库 snapshot，避免订阅前的状态变化丢失。
4. stream 返回当前任务详情、最近日志和 `log_cursor`，之后继续推送状态 patch 和日志增量。
5. 任务完成或失败后，stream 保留短暂 final flush 窗口，再发送 `complete` 并自然结束。

## 结构化事件

后端事件由 `backend/internal/websocket/sync_task_events.go` 定义。事件序列 `sequence` 在单个 `sync_id` 内单调递增，前端用它丢弃同一任务的旧事件。跨任务或目录级排序不应只依赖 `sequence`，可结合 `event_time`。

| 字段 | 含义 |
| --- | --- |
| `sync_id` | 真实同步记录 ID |
| `sync_path_id` | 同步目录 ID |
| `status` / `sub_status` | 同步任务状态和子状态 |
| `total` / `new_strm` / `new_meta` / `new_upload` | 运行统计 |
| `finish_at` | 完成或失败时间 |
| `net_file_start_at` / `net_file_finish_at` | 处理网盘文件阶段时间 |
| `local_file_start_at` / `local_file_finish_at` | 处理本地文件列表阶段时间 |
| `log_path` | 后端根据 `sync_id` 生成的日志相对路径 |
| `sequence` | 单个 `sync_id` 内的事件序列 |
| `event_time` | 后端事件产生时间 |
| `created_at` / `updated_at` | 记录时间戳 |
| `local_path` / `remote_path` | 同步路径展示字段 |
| `fail_reason` | 失败原因 |
| `deleted` | 记录删除标记 |
| `resync_reason` | 要求前端重新同步的原因 |

旧事件 `strm_sync_task_start` 和 `strm_sync_task_complete` 保留。迁移后的页面优先消费新结构化事件，旧事件只作为后台校准兜底。

全局事件还包含：

- `strm_sync_task_queued`：STRM 同步任务加入等待队列时尝试广播，数据包含 `sync_path_id`、`is_running=1` 和 `task_type`。该事件在任务交付给队列处理器前尝试发送，避免晚到的等待事件覆盖已经运行中的状态；当 WebSocket 广播缓冲区已满时会丢弃该即时提示，不阻塞同步队列。同步目录页收到后立即显示“等待中”，不需要等待页面刷新。

## Stream 消息

详情 stream 位于鉴权路由组内，路径为 `GET /api/sync/tasks/:id/stream`。WebSocket 消息包含 `type`、`version`、`sync_id`、`server_time`，状态类消息还会带 `sequence`。

| type | 说明 |
| --- | --- |
| `snapshot` | 当前任务详情、最近日志、`log_cursor`、`log_path` |
| `task_patch` | 状态、子状态和统计字段变化 |
| `log_append` | 日志增量，带日志 `cursor` |
| `complete` | 完成、失败或记录删除的收尾消息 |
| `heartbeat` | 连接保活 |
| `error` | 服务端错误 |
| `resync_required` | 前端应静默重连并重新获取 snapshot |

`snapshot.log_cursor` 和 `log_append.cursor` 是日志游标，不等同于任务事件 `sequence`。

## 前端刷新边界

- 同步任务详情使用 `useSyncTaskStream(syncId)`，不再拼接 `loadTaskInfo` 和通用 `AppLogViewer`。
- 详情页首次进入可以显示 loading；收到 snapshot 后，任务状态和日志都在原位置更新，不重建日志组件。
- `SyncTaskLogPanel` 只在用户处于“跟随最新”状态时自动跟随顶部，用户查看历史日志时不会抢滚动。
- 同步记录页按 `sync_id` 和 `sequence` patch 当前行；只有 `sync_task_created` 可以在第一页插入新记录，缺失当前页的 `sync_task_updated` 不插入，避免进度事件污染分页排序和总数。
- 同步记录和详情页根据 `new_strm`、`new_meta` 与任务状态派生媒体库刷新相关提示；前端只展示是否存在刷新相关变更，不声明后端已提交刷新任务。
- 同步目录页按 `sync_path_id` 更新运行状态，并按 `sync_id` 去重事件；同一目录的新任务不会被上一任务的 sequence 压掉。

## 日志 tailer

通用日志和同步任务日志都使用 `logstream.GlobalManager`。manager 按日志文件绝对路径复用共享 tailer，同一个文件的多个 WebSocket 客户端不会各自启动独立 tail goroutine。

日志读取规则：

- 日志文件不存在时，历史读取返回空日志和 cursor `0`。
- 同步任务详情日志按“最新在上”展示；实时增量插入顶部，快照日志会按时间倒序显示。工具栏提供连接、断开、清空和下载日志，下载仍复用 `/api/logs/download`。
- 通用实时日志 WebSocket 建连时只读取文件末尾 cursor，历史内容由 `/api/logs/old` 加载，避免大日志文件建连时全量扫描。
- tailer 发现日志文件缺失、截断或订阅者缓冲满时，会发送 `resync_required`。
- tailer 检测文件大小小于 cursor 时重置 cursor，并要求前端重新同步。
- 半行日志会暂存在 tailer 中，直到收到换行后再发送完整日志；半行超过 1 MiB 时会清空缓存并发送 `resync_required`。
- 按 cursor 补读最多读取有限行数，超过补读窗口时发送 `resync_required`。

## 兼容和暂不迁移范围

- 首页运行日志继续使用通用日志组件读取 `app.log`，不接入同步任务详情 stream。
- 通用日志接口 `/api/logs/old`、`/api/logs/ws`、`/api/logs/download` 保留。
- 刮削记录、上传队列、下载队列和备份记录继续使用各自现有刷新机制。
- 旧同步事件保留，迁移期间可继续作为兜底校准信号。

## 主要边界

- `task_id` 仍保持旧事件原语义，新事件必须使用 `sync_id` 表示真实同步记录 ID。
- 断线、服务重启或 `resync_required` 后，前端重新建立 stream 并获取 snapshot，不假设事件连续。
- WebSocket 鉴权依赖当前站点凭据；原生 WebSocket 不走 axios 拦截器。
- 运行中任务完成后可能还有最后几行日志，stream 使用 final flush 窗口降低丢日志概率。
- 只有已完成或失败的同步记录允许手动删除；删除同步记录会发布 `sync_task_deleted`。刷新后打开已删除记录时，stream 返回 `complete` + `deleted`，详情页展示删除状态并停止重连。
- 删除同步目录不会级联删除同步历史；同步记录详情在关联同步目录不存在时仍以历史记录和日志为准。
- 如果状态事件和日志写入顺序不完全一致，以 snapshot 和数据库状态作为恢复后的事实来源。
