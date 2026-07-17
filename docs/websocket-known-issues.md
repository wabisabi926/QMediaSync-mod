# WebSocket 实现维护说明

本分支 `backup/ws-before-sse-migration-20260717` 用于保留 SSE 迁移前的
WebSocket 实现和功能。本说明记录截至 2026-07-17 已确认的实现风险，供维护
此分支或回退时参考；本文不改变现有协议、路由或用户行为。

## 当前通道

- `/api/events/ws`：全局队列、刮削和同步记录变更通知。
- `/api/logs/ws`：指定日志文件的新增日志行。
- `/api/sync/tasks/:id/stream`：同步任务 snapshot、状态 patch 和日志增量。

所有通道目前只由服务端向浏览器推送；启动、停止、暂停和删除等操作仍通过
受鉴权、CSRF 保护的 HTTP API 完成。

## 已确认风险

### 全局事件广播可能阻塞业务调用

`backend/internal/websocket/event_hub.go` 的 `BroadcastEvent()` 直接向容量为
256 的 `broadcast` channel 发送数据。当事件中心处理不及或该 channel 已满时，
发送方会阻塞。上传、下载、同步或刮削业务调用该函数时，可能被浏览器实时通知
反向拖慢。

`TryBroadcastEvent()` 已使用非阻塞发送，但目前不是所有全局事件都使用该函数。
维护本分支时，新增高频事件应优先评估是否允许丢失通知，并使用非阻塞投递或
独立的背压策略。

### 慢客户端移除的锁处理不稳健

事件中心的 `Run()` 在遍历 `clients` 时先持有 `RLock()`；发现慢客户端后释放
读锁、获取写锁删除并关闭 channel，再重新获取读锁继续遍历。这个锁切换发生在
同一次 map 遍历过程中，依赖并发调用时序，难以证明不会与注册或注销竞争。

维护时应将订阅、发送、删除和关闭的生命周期收敛到一个明确的同步边界，避免在
遍历 map 的中途释放保护该 map 的锁。

### WebSocket Origin 校验被显式放开

`event_websocket.go`、`log_websocket.go` 与 `sync_task_stream.go` 的
`websocket.Upgrader.CheckOrigin` 均直接返回 `true`。当部署使用 Cookie 会话时，
这会放宽跨站 WebSocket 连接限制。

维护本分支时应至少限制为当前站点 Origin，或在明确支持跨站部署时维护严格的
Origin 白名单和对应的 Cookie `SameSite` 策略。

### 前端全局连接不会在无人监听时关闭

`frontend/src/composables/useWebSocket.ts` 以模块级单例维持连接。最后一个事件
监听器注销后，该连接仍会保持；页面在 `KeepAlive` 之间切换或所有相关页面离开后，
仍可能消耗一个 WebSocket 连接。

建议在监听器集合为空时主动关闭连接、清理重连 timer，并在下一次订阅时再建立连接。

### 全局与同步任务连接的重连次数有限

前端全局事件连接和同步任务详情连接均在连续失败 5 次后停止重连。临时断网、服务
重启或代理短暂不可用超过该窗口时，页面不会自行恢复；用户需要刷新页面或重新进入
页面。

若继续维护 WebSocket 方案，建议改为带上限间隔与抖动的持续重连，并在重连成功后
拉取 HTTP snapshot，避免断线窗口内遗漏状态变更。

## 已有恢复边界

全局事件是轻量刷新通知，日志和同步任务详情也不承诺可靠投递。出现断线、慢客户端
或事件丢失时，应以既有 HTTP 列表、历史日志和同步任务 snapshot 重新加载后的状态
为准，而不是把 WebSocket 消息当作持久化事件日志。
