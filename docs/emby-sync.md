# Emby 同步维护说明

本文说明 QMediaSync 中 Emby 刷新目标和同步条目的职责边界，以及全量同步、增量同步、Webhook 单条同步和运行状态字段。业务时间字段策略见 [数据库 - 时间字段策略](database.md#时间字段策略)，Cron 表达式边界见 [请求校验约定 - Cron 表达式边界](validation.md#cron-表达式边界)。

实现细节以 `backend/internal/emby`、`backend/internal/models`、`backend/internal/synccron` 和 `frontend/src/components/AppEmbySettings.vue` 为准。

## 两条链路

Emby 相关任务分为两条独立链路，不能混用。

### 链路 A：通知 Emby 刷新

含义：QMediaSync 通知 Emby 刷新已有 item 或重新扫描它自己的媒体库。

触发场景：

- STRM 文件生成或更新完成。
- 元数据文件生成或更新完成。
- 同步目录关联的 Emby 媒体库需要重新扫描。

当前实现：

- 传统同步目录刷新调用 `models.RequestEmbyLibraryRefreshBySyncPathId(...)`。
- 上传完成、远端已存在等非 Webhook STRM 任务生成 STRM 后调用 `models.RequestEmbyRefreshBySyncFile(...)`，优先解析 item 级刷新目标。
- STRM Webhook 只有 `refresh_emby=true` 且 STRM 变更或新增元数据下载任务时才解析刷新目标；批量和目录扫描先累计子任务目标，父任务全部子任务完成或失败后调用 `models.RequestEmbyRefreshTargets(...)` 统一提交。
- 已有关联的 Movie、Video、Episode 刷新对应 item；同季新增剧集没有自身 Episode 关联时优先刷新已有 Season，缺少 Season 时刷新 Series。
- 本地索引无法定位 item 时，会按 STRM 本地路径调用 Emby `/Items?Path=...` 做一次兜底查询。
- 仍无法定位可靠 item 时，回退同步目录关联媒体库刷新。
- 创建或合并 `EmbyLibraryRefreshTask`。
- 刷新任务创建、更新或下载事件批量落库后，会把全局最近到期 timer 调度到最早的 `pending.refresh_after_at`。
- 并发调度时，如果一个较旧的查询结果晚于较新的更早 timer 返回，旧结果不能取消或覆盖已有更早 timer；最多提前唤醒检查一次。
- 如果调度时发现已有到期且未按当前 `refresh_after_at` 检查过的 pending 任务，会立即触发一次检查。
- timer 到期后通过原有检查通道唤醒协调器；60 秒 ticker 仍保留为兜底。
- 协调器等待相关下载任务和 STRM 同步任务结束；如果到期时仍有下载任务，会继续等待后续下载事件或 60 秒 ticker 兜底检查。
- 最后按任务目标调用 Emby item 刷新或媒体库刷新接口。

全局 timer 只负责到点唤醒检查，不直接决定是否刷新。多个刷新任务有各自的 `refresh_after_at` 和下载任务状态，全局 timer 只指向最早到期的一条；每次检查仍按任务独立判断是否 ready。

- item 定向刷新接口：

```http
POST /emby/Items/{itemId}/Refresh?Recursive=<true|false>
```

- 媒体库刷新接口：

```http
POST /emby/Items/{libraryId}/Refresh
```

职责边界：

- 只调用 Emby 刷新接口。
- 不拉取 Emby item。
- 不写入 `emby_media_items`。
- 不建立 PickCode 和 `sync_files` 关联。
- 不和链路 B 串成一个长任务。

前端状态文案为“刷新 Emby 媒体库”，后端状态值为 `refresh_library`。

### 链路 B：同步 Emby 条目到本地

含义：QMediaSync 从 Emby 拉取媒体条目，写入本地数据库，用于建立 Emby item、PickCode 和同步文件之间的关联。

触发场景：

- 手动全量同步。
- Cron 定时增量同步。
- Webhook 新增或修改事件触发单条同步。
- 低频全量校验或重建本地索引。

职责边界：

- 只维护 QMediaSync 本地索引。
- 不调用 Emby 媒体库刷新接口。
- 不等待 STRM 生成流程。
- 不把“通知 Emby 扫库”和“同步 Emby 条目”串成一个长任务。

前端状态文案：

| 状态值 | 展示文案 |
| --- | --- |
| `idle` | 空闲 |
| `full` | 全量同步 Emby 条目 |
| `incremental` | 增量同步 Emby 条目 |
| `webhook` | Webhook 单条同步 Emby 条目 |
| `refresh_library` | 刷新 Emby 媒体库 |

## 同步架构

推荐运行模式是“Webhook 实时单条处理 + 定时增量兜底 + 低频全量校验”。

### 分页流式处理

全量同步和增量同步都复用分页拉取逻辑：

- 每页 `Limit` 默认取 `100`。
- 每取到一页就逐条处理 item，不把所有媒体项聚合成一个大切片。
- 每个 item 处理时完成 PickCode 提取、`emby_media_items` upsert 和 `emby_media_sync_files` 关联维护。
- worker pool 可以保留，但需要限制并发，避免对 Emby 和数据库造成突发压力。
- 不再对每个 item 固定 sleep；如需限速，应限制分页请求频率或使用可配置限速。

### 手动全量同步

手动同步用于低频校验或重建本地索引。

流程：

1. 生成本轮 `sync_run_id`。
2. 按媒体库分页拉取 Emby item。
3. 每个 item upsert 时写入 `last_seen_sync_run` 和 `last_seen_at`。
4. 某个媒体库完整同步成功后，只清理该媒体库内 `last_seen_sync_run != 当前批次` 的旧条目。
5. 如果某个媒体库拉取失败，该媒体库不执行旧数据清理，避免误删。

全量同步不再维护巨大的有效 item ID 列表，也不再对每个 item 固定 sleep。

### 定时同步

Cron 默认表达式为 `0 * * * *`，含义是每小时整点执行一次。

当 `enable_daily_first_full_sync=1` 时，每天第一次定时同步会执行全量同步；判断依据是 `last_full_sync_at` 是否属于服务器本地时区的当天。全量同步成功后会更新 `last_full_sync_at`，当天后续定时同步执行增量同步。如果当天首次全量失败，`last_full_sync_at` 不会推进，下一次定时同步仍会继续尝试全量。

当 `enable_daily_first_full_sync=0` 时，Cron 每次都执行增量同步。

保存 Emby 配置时，如果兼容客户端没有提交 `enable_daily_first_full_sync` 字段，后端会保留当前已有配置；只有显式提交 `0` 或 `1` 时才会修改该开关。

### 定时增量同步

请求使用 Emby `/Items` 的 `MinDateLastSaved` 参数。OpenAPI 中该参数类型为 `string`、格式为 `date-time`，分页参数为 `StartIndex` 和 `Limit`：

```http
GET /emby/Items
?ParentId=<libraryId>
&Recursive=true
&IncludeItemTypes=Movie,Video,Episode
&Fields=DateCreated,DateModified,ParentId,PremiereDate,MediaStreams,Path,MediaSources,SeriesId,SeasonId,SeriesName,SeasonName,IndexNumber,ParentIndexNumber
&MinDateLastSaved=<cursor_minus_overlap_rfc3339>
&SortBy=DateLastSaved
&SortOrder=Descending
&StartIndex=0
&Limit=100
```

响应示例：

```json
{
  "TotalRecordCount": 1,
  "Items": [
    {
      "Id": "10001",
      "Name": "示例影片",
      "Type": "Movie",
      "DateCreated": "2026-06-01T12:00:00.0000000Z",
      "DateModified": "2026-06-20T08:30:00.0000000Z",
      "HasPath": true,
      "MediaSources": [
        {
          "Id": "mediasource_10001",
          "HasPath": true,
          "PathPrefix": "http://qmediasync:12333/"
        }
      ]
    }
  ]
}
```

游标规则：

- 本轮开始时记录 `scan_started_at = now.UTC().Unix()`。
- 查询时使用 `last_saved_cursor_at - 600 秒`，初始或小于 overlap 时使用 Unix 0。
- 本轮内用 item ID 去重。
- 全部媒体库成功后更新 `last_incremental_sync_at`、`last_saved_cursor_at`、`last_processed_count` 和 `last_sync_time`。
- 任一媒体库失败时不推进 `last_saved_cursor_at`。
- 不依赖响应中的 `DateLastSaved` 推进游标，因为实际响应不返回该字段。
- 增量同步不做全局删除。

过滤边界：

- `MinDateLastSaved=2999-01-01T00:00:00Z` 这类未来时间应返回空集合。
- 空集合只表示本轮没有变更，不代表本地已有数据应被删除。
- 因响应不返回 `DateLastSaved`，不能用最后一条 item 的时间推进游标。

### Webhook 单条同步

Webhook 新增或修改事件只按 item ID 查询单条并 upsert：

```http
GET /emby/Items?Ids=<itemId>&Fields=DateCreated,DateModified,Path,MediaSources,ParentId,SeriesId,SeasonId,SeriesName,SeasonName,IndexNumber,ParentIndexNumber
```

响应示例：

```json
{
  "TotalRecordCount": 1,
  "Items": [
    {
      "Id": "10001",
      "Name": "示例影片",
      "Type": "Movie",
      "DateCreated": "2026-06-01T12:00:00.0000000Z",
      "DateModified": "2026-06-20T08:30:00.0000000Z",
      "HasPath": true,
      "MediaSourceCount": 1
    }
  ]
}
```

如果需要用户态字段，可以改用：

```http
GET /emby/Users/{UserId}/Items/{itemId}?Fields=DateCreated,DateModified,Path,MediaSources
```

兼容注意：

- `/Items/{itemId}` 在部分 Emby 服务上可能返回 `404`，不能作为唯一单条详情路径。
- `/Items?Ids=<itemId>` 更适合无用户上下文的单条查询。
- `/Users/{UserId}/Items/{itemId}` 适合需要用户态字段时使用。

规则：

- 返回单条 Movie、Video 或 Episode 时写入本地 `emby_media_items`。
- 写入前会通过 `/Items/{itemId}/Ancestors` 和 `/Library/VirtualFolders` 解析真实所属媒体库，不能把 Episode 的 `ParentId` 当作媒体库 ID。
- 当 `sync_all_libraries=0` 时，解析出的媒体库 ID 不在 `selected_libraries` 中会跳过本次单条同步。
- 解析到 PickCode 时建立 `emby_media_sync_files` 关联。
- 空结果或不支持的类型只记录日志，不触发全量同步。
- Webhook 单条同步不推进 `last_saved_cursor_at`。

Webhook 删除事件只删除本地索引和关联：

- Movie、Video、Episode：按 `item_id` 删除本地 item 和关联。
- Season：按 `season_id` 删除本地 item 和关联。
- Series：按 `series_id` 删除本地 item 和关联。
- 如果启用联动删除网盘文件，会先执行原有网盘删除逻辑，然后清理本地索引。
- 删除事件不触发全量同步，也不调用刷新媒体库接口。

## 同步运行状态

状态接口返回的核心字段：

| 字段 | 含义 |
| --- | --- |
| `is_running` | 是否有 Emby 条目同步任务运行 |
| `sync_mode` | 当前或最近一次模式 |
| `started_at` | 当前任务开始时间 |
| `last_sync_time` | 最近一次成功同步时间 |
| `last_success_sync_mode` | 最近一次成功同步模式 |
| `last_full_sync_at` | 最近一次成功全量同步时间 |
| `last_incremental_sync_at` | 最近一次成功增量同步时间 |
| `last_saved_cursor_at` | 增量同步游标 |
| `last_processed_count` | 最近一次处理数量 |
| `last_error` | 最近一次失败原因 |
| `total_items` | 本地已同步 Emby item 数 |

如果已有任务运行，定时任务直接跳过，不排队堆积。手动全量同步遇到运行中的任务时，前端应提示稍后再试。

程序启动并完成数据库迁移后，会清理上次进程异常退出遗留的 `is_running=true` 状态，避免全量、增量和 Webhook 单条同步长期被旧运行标记阻塞。该清理只复位 `is_running`、`sync_mode` 和 `started_at`，并写入 `last_error` 说明原因；不会推进或清空 `last_sync_time`、`last_full_sync_at`、`last_incremental_sync_at` 和 `last_saved_cursor_at`。

## 升级和迁移说明

新增字段通过 GORM AutoMigrate 补齐，旧数据默认值保持兼容。

新增或扩展的字段：

- `emby_config.last_full_sync_at`
- `emby_config.last_incremental_sync_at`
- `emby_config.last_saved_cursor_at`
- `emby_config.last_processed_count`
- `emby_config.last_success_sync_mode`
- `emby_config.last_error`
- `emby_config.is_running`
- `emby_config.sync_mode`
- `emby_config.started_at`
- `emby_config.enable_daily_first_full_sync`
- `emby_media_items.last_seen_sync_run`
- `emby_media_items.last_seen_at`

升级后如果 `last_saved_cursor_at=0`，建议先手动执行一次全量同步，用于建立本地索引和全量批次标记。默认启用每日首次定时全量同步，之后 Cron 会在每天首次成功全量后按增量同步兜底变更。

## 验证命令

后端：

```bash
(cd backend && go test ./...)
(cd backend && go vet ./...)
```

前端：

```bash
(cd frontend && pnpm exec vitest run test/components/AppEmbySettings.sync-status.test.ts)
(cd frontend && pnpm exec vitest run test/utils/timeUtils.test.ts)
(cd frontend && pnpm run type-check)
(cd frontend && pnpm run build)
```
