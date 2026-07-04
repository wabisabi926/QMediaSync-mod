# 上传和 STRM 后处理流程

本文说明 QMediaSync 当前上传增强、目录监控上传和 STRM 生成后处理的运行边界。

## 115 上传增强

115 上传任务先执行 `/open/upload/init`。如果接口返回秒传成功，任务会直接进入完成态并记录 `upload_result=rapid_upload`；如果未命中秒传且启用了秒传等待策略，任务会按 `upload_rapid_wait_interval_seconds` 重复尝试 init，直到命中秒传或达到 `upload_rapid_wait_timeout_seconds`。`upload_rapid_wait_min_size` 控制进入等待策略的最小文件大小，`upload_rapid_wait_force_size` 控制必须等待到超时的大文件阈值；等待超时后是否跳过真实上传由 `upload_rapid_wait_skip_upload` 控制。

非秒传上传使用 OSS multipart。默认 part size 为 `32 MiB`；当文件按该大小切分会超过 `9999` 个 part 时，part size 会按文件大小动态放大并向上取整到 `1 MiB`。首次创建 `upload_sessions` 后会持久化 part size、OSS `upload_id`、本地文件签名、115 调度字段和已上传进度，后续重试或进程重启恢复时必须复用这些 checkpoint。

断点续传同时依赖 115 调度层和 OSS 数据层。恢复时先调用 115 `/open/upload/resume`，再用 OSS `ListParts` 查询已有分片并跳过已完成 part；如果本地文件大小、mtime、SHA1 或快速签名变化，旧 session 会标记为 `aborted`，任务失败而不是误用旧 checkpoint。

OSS `CompleteMultipartUpload` 完成后，必须带回 115 init 返回的 `callback` / `callback_var`。如果 115 callback 响应 `state=false`、缺少远端文件 ID、缺少 PickCode 或响应无法解析，任务不会视为上传成功，也不会创建后续 STRM 生成任务；错误会写入上传任务和 `upload_sessions.complete_callback_error` 供排查。

## 目录监控上传

目录监控上传规则绑定一个同步目录，只支持 115 Open API 上传目标。程序启动后会加载启用的规则；规则停用或程序退出时，后台 watcher / polling 会停止。

规则接口：

- `GET /api/directory-upload/rules`：查询规则列表，可用 `sync_path_id` 过滤。
- `POST /api/directory-upload/rules`：创建规则。
- `PUT /api/directory-upload/rules/:id`：更新规则。
- `DELETE /api/directory-upload/rules/:id`：删除规则。
- `POST /api/directory-upload/rules/:id/status`：启用或停用规则，请求体为 `{"enabled": true}`。
- `POST /api/directory-upload/rules/:id/scan`：手动触发补偿扫描，返回本次加入稳定性队列的候选文件数。

监控模式：

- `auto`：优先使用 `fsnotify` watcher；初始化失败时自动回退到 polling。
- `watcher`：强制使用 `fsnotify` watcher，初始化失败则规则启动失败。
- `polling`：按 `rescan_interval_seconds` 周期递归扫描。

启动补偿扫描由 `startup_scan_enabled` 控制。补偿扫描只把候选视频文件加入稳定性队列，不直接创建上传任务。

## 稳定性和去重

目录监控发现文件后，会先进入稳定性队列。稳定性签名为文件 `size + mtime`；签名变化会重置稳定计数。只有文件满足 `stability_seconds` 和 `stability_required_count` 后，才会创建上传任务。

同一规则下，`monitor_path + relative_path + signature` 会按 `processed_cache_ttl_seconds` 做内存 TTL 去重，避免 create / write 多事件重复创建任务。TTL 过期且文件签名变化后允许再次处理。

默认忽略隐藏文件、`.part`、`.tmp`、`.download` 文件，以及规则中的 `ignore_patterns`。

## 上传任务和远端已存在

目录监控上传只创建 `db_upload_tasks.source = directory_monitor` 的上传任务，真实上传仍由全局上传队列执行。任务会写入 `sync_path_id`、`relative_path`、`remote_file_id` 和 `remote_path_id`，因此会出现在上传队列页面。

创建任务前会检查远端同目录同名文件。只有远端文件 SHA1 和大小都与本地文件一致时，才把上传任务直接标记为 `completed`，`upload_result = remote_exists`，并创建后续 STRM 生成任务。该行为是远端已存在跳过，不是断点续传。

普通上传成功、秒传成功或断点续传完成后，由上传任务统一创建 STRM 生成任务。`upload_result = skipped_after_rapid_wait` 不会创建 STRM 生成任务，也不会触发源文件删除。

## STRM 后处理和源文件删除

STRM 生成 worker 会读取 `strm_generation_tasks`，复用同步目录配置写入或确认 STRM。STRM 新增或更新后，会优先提交 Emby item 级定向刷新；定位不到可靠 item 时回退同步目录关联媒体库刷新。

如果同一远端文件发生移动或重命名，服务会用 `file_id` / `pick_code` 查找旧 `SyncFile`。新 STRM 写入成功后，只精确删除旧记录里的 `local_file_path`，不会按文件名模糊删除其他 `latest` 或同名文件。旧 STRM 删除失败时，新 STRM 会保留，任务记录失败原因，便于从 `strm_generation_tasks.last_error` 和应用日志排查。

目录监控规则的 `delete_source_after_success` 默认关闭。开启后，也必须同时满足以下条件才会删除本地源文件：

- 上传任务来源为 `directory_monitor`。
- 上传任务状态为 `completed`。
- 上传结果为 `rapid_upload`、`multipart_uploaded` 或已确认签名的 `remote_exists`。
- 关联 `StrmGenerationTask` 状态为 `completed`。
- 源文件路径仍在规则 `monitor_path` 内。

删除源文件成功后，程序会从源文件所在目录向上删除空目录，但不会删除 `monitor_path` 根目录。清理失败只记录到上传任务的 `source_cleanup_status` 和 `source_cleanup_error`，不会回滚远端文件或已生成的 STRM。

## STRM Webhook

`POST /api/strm/webhook` 用于外部程序请求生成 STRM。该接口不使用浏览器登录态，只接受 API Key 鉴权，支持 `X-API-Key` header 和 `?api_key=` 查询参数。无效 API Key 返回 HTTP `401`。

请求体支持三类动作：

- `action=file`：创建单文件 STRM 生成任务。
- `action=batch_files`：按 `items` 批量创建单文件任务，单项失败不会影响其他合法项入队。
- `action=directory_scan`：创建目录扫描父任务，只记录目录任务，不在请求内同步展开完整目录。

`action` 为空时，后端会按字段自动判断：有 `items` 时视为批量文件，有 `directory_id` 或 `directory_path` 时视为目录扫描，否则视为单文件。

文件级请求必须提供 `sync_path_id`，并至少提供 `file_id`、`pick_code`、`path + file_name` 中的一组定位信息。目录级请求必须提供 `directory_id` 或 `directory_path`。`path` 和 `directory_path` 都表示 115 远端路径；如果传入路径，必须位于对应同步目录的 `remote_path` 下。

接口禁止通过 `local_path` 指定本地写入位置。本地 STRM 路径只能由 `sync_path_id` 对应同步目录计算。合法请求只创建 `strm_generation_tasks`，不会在 HTTP 请求内直接写 STRM，也不会触发 Emby 条目同步；后续仍由 STRM worker 生成 STRM，并在 STRM 变更后提交 Emby item 定向刷新或媒体库回退刷新。

单文件请求示例：

```json
{
  "sync_path_id": 1,
  "action": "file",
  "file_id": "115-file-id",
  "pick_code": "pickcode",
  "path": "/电影/示例影片",
  "file_name": "示例影片.mkv",
  "size": 1073741824,
  "sha1": "VIDEO_FILE_SHA1"
}
```

批量文件请求示例：

```json
{
  "sync_path_id": 1,
  "action": "batch_files",
  "items": [
    {
      "file_id": "115-file-id-1",
      "pick_code": "pickcode1",
      "path": "/剧集/示例剧/S01",
      "file_name": "S01E01.mkv"
    },
    {
      "file_id": "115-file-id-2",
      "pick_code": "pickcode2",
      "path": "/剧集/示例剧/S01",
      "file_name": "S01E02.mkv"
    }
  ]
}
```

目录扫描请求示例：

```json
{
  "sync_path_id": 1,
  "action": "directory_scan",
  "directory_id": "115-directory-id",
  "directory_path": "/剧集/示例剧/S01"
}
```
