# 上传和 STRM 后处理流程

本文说明 QMediaSync 当前上传增强、目录监控上传和 STRM 生成后处理的运行边界。

## 115 上传增强

115 上传任务先执行 `/open/upload/init`。115 官方秒传返回只保证包含新增 `file_id`，不返回 mtime；因此秒传成功后系统会按 `file_id` 查询远端文件详情，补齐 PickCode、SHA1、大小和 mtime，并记录 `upload_result=rapid_upload`。`strm_sync` 上传需要用远端 mtime 同步本地元数据文件，详情查询失败会让任务失败；目录监控上传可用 init 返回的 `file_id` 兜底完成，后续 STRM worker 仍可按 `file_id` 补齐文件详情。如果未命中秒传且启用了秒传等待策略，任务会按 `upload_rapid_wait_interval_seconds` 重复尝试 init，直到命中秒传或达到 `upload_rapid_wait_timeout_seconds`。`upload_rapid_wait_interval_seconds` 是两次 init 之间的重试间隔，`upload_rapid_wait_timeout_seconds` 是最大等待时长；最后一次等待会按剩余超时时间裁剪，不会因为间隔更长而超过最大等待时长。`upload_rapid_wait_min_size` 控制进入等待策略的最小文件大小，`upload_rapid_wait_force_size` 控制必须等待到超时的大文件阈值；等待超时后是否跳过真实上传由 `upload_rapid_wait_skip_upload` 控制。

非秒传上传使用 OSS multipart。初始化 OSS multipart 时会带 `sequential=1`；真实非秒传任务验证显示，该参数配合 115 callback 原样透传后，OSS 会在完成对象上返回可供 115 校验的最终 SHA1。默认 part size 为 `32 MiB`；当文件按该大小切分会超过 `9999` 个 part 时，part size 会按文件大小动态放大并向上取整到 `1 MiB`。首次创建 `upload_sessions` 后会持久化 part size、OSS `upload_id`、本地文件签名、115 调度字段和已上传进度，后续重试或进程重启恢复时必须复用这些 checkpoint。

断点续传同时依赖 115 调度层和 OSS 数据层。恢复时先调用 115 `/open/upload/resume`，再用 OSS `ListParts` 查询已有分片并跳过已完成 part；如果本地文件大小、mtime、SHA1 或快速签名变化，旧 session 会标记为 `aborted`，任务失败而不是误用旧 checkpoint。如果 OSS 返回 `NoSuchUpload`、`InvalidUploadId` 等明确 checkpoint 失效错误，任务会清空旧 `upload_id`、已上传字节和分片进度，将恢复状态标记为 `session_expired_restarted`，并在同一次任务中复用当前 115 调度结果创建新的 OSS multipart。

OSS `CompleteMultipartUpload` 完成后，必须带回 115 init 返回的 `callback` / `callback_var`。官方 `callback` 可能是单个对象或对象数组；对象数组会取第一个回调配置。传给 OSS 时只把 115 返回的 callback JSON 原样 Base64 编码为 `x-oss-callback` / `x-oss-callback-var`，由 OSS 处理 `${bucket}`、`${object}`、`${size}`、`${sha1}` 和 `${x:...}` 占位符；后端不要提前展开 `callbackBody`，也不要向 `callback_var` 增加非 `x:` 字段。本地 SHA1 不传给 OSS multipart，`callbackBody` 中的 `${sha1}` 以 OSS 完成对象后返回的最终 SHA1 为准。如果 115 callback 响应 `state=false`、缺少远端文件 ID、缺少 PickCode 或响应无法解析，任务不会视为上传成功，也不会创建后续 STRM 生成任务；错误会写入上传任务和 `upload_sessions.complete_callback_error` 供排查。

## 目录监控上传

目录监控上传规则绑定一个同步目录，只支持 115 Open API 上传目标。程序启动后会加载启用的规则；规则停用或程序退出时，后台 fsnotify / polling 会停止。

规则接口：

- `GET /api/directory-upload/rules`：查询规则列表，可用 `sync_path_id` 过滤。
- `POST /api/directory-upload/rules`：创建规则。
- `PUT /api/directory-upload/rules/:id`：更新规则。
- `DELETE /api/directory-upload/rules/:id`：删除规则。
- `POST /api/directory-upload/rules/:id/status`：启用或停用规则，请求体为 `{"enabled": true}`。
- `POST /api/directory-upload/rules/:id/scan`：手动触发扫描，返回本次加入稳定性队列的候选文件数；规则停用时会拒绝执行。

监控模式：

- `auto`：自动（推荐），优先使用 fsnotify 实时发现；初始化失败时自动回退到 polling 查漏。
- `fsnotify`：性能模式，强制使用 fsnotify，初始化失败则规则启动失败。
- `polling`：兼容模式，按内置 30 秒周期递归扫描。

启动查漏由 `startup_scan_enabled` 控制。查漏扫描默认只把候选视频文件加入稳定性队列；规则 `upload_metadata=true` 时，也会纳入当前同步目录配置中的元数据扩展名文件。视频扩展名和元数据扩展名都按同步目录自定义配置优先、为空回退全局 STRM 设置的规则解析。`recursive=false` 时，查漏扫描和 fsnotify 事件都只处理监控根目录下的文件，新建子目录不会被加入 watcher。扫描不直接创建上传任务。补偿扫描间隔为代码内置 30 秒，不提供页面或接口配置。

## 稳定性和去重

目录监控发现文件后，会先进入稳定性队列。稳定性签名为版本化源文件 fingerprint，格式为 `v1:size:mtime_ns`；签名只包含文件大小和纳秒级 mtime，不包含 ctime、inode 或文件内容 hash。签名变化会重置稳定计数。稳定性检查间隔为内置 2 秒，文件需要在内置 15 秒稳定窗口内保持签名不变，并连续 3 次检查不变后，才会创建上传任务。这些稳定性参数不提供页面或接口配置。

同一规则下，监控规则、相对路径和 `source_fingerprint` 会按 `processed_cache_ttl_seconds` 做内存 TTL 去重，避免 create / write 多事件重复创建任务。TTL 过期且文件 fingerprint 变化后允许再次处理。创建上传任务前还会按 `source=directory_monitor + local_full_path + pending/uploading` 查询数据库；已有未完成任务时会跳过重复入队，覆盖服务重启、轮询重复发现和大文件长时间上传场景。

默认忽略隐藏文件、`.part`、`.tmp`、`.download` 文件，以及规则中的 `ignore_patterns`。规则列表接口会把持久化的忽略规则解析为 `ignore_patterns` 数组返回，避免页面保存其他目录监控配置时丢失已有忽略规则。

## 上传任务和远端已存在

目录监控上传只创建 `db_upload_tasks.source = directory_monitor` 的上传任务，真实上传仍由全局上传队列执行。任务会写入 `sync_path_id`、`relative_path`、`source_fingerprint`、`local_mtime_ns`、`remote_file_id` 和 `remote_path_id`，其中 `source_fingerprint` 使用 `v1:size:mtime_ns`。因此任务会出现在上传队列页面。

创建任务前会检查远端同目录同名文件。只有远端文件大小和 SHA1 都与本地文件一致时，才把上传任务直接标记为 `completed`，`upload_result = remote_exists`，并创建后续 STRM 生成任务。该行为是远端已存在跳过，不是断点续传。

同名文件大小或 SHA1 不一致时，按 `overwrite_mode` 处理：

- `skip_same`：跳过本地文件，不创建上传任务，不删除远端文件。
- `fail_conflict`：停止处理并记录错误，不创建上传任务。
- `replace_conflict`：先删除远端同名文件，再创建新的上传任务。

普通上传成功、秒传成功或断点续传完成后，由上传任务统一创建 STRM 生成任务。`upload_result = skipped_after_rapid_wait` 不会创建 STRM 生成任务，也不会触发源文件删除。

## STRM 后处理和源文件删除

STRM 生成 worker 会读取 `strm_generation_tasks`，复用同步目录配置写入或确认 STRM。该后处理只创建或更新 `SyncFile` 和 STRM 文件，不创建 `syncs` 同步记录，也不向同步目录队列添加“等待中”任务；完整同步记录只由手动同步、定时同步等 STRM 同步入口创建。文件级任务会先比较已有 STRM 内容；确认需要更新后直接写入新 STRM，不再重复比较，因此同一次后处理只输出一次 PickCode、路径或用户 ID 差异日志。文件级任务在文件名、路径、父目录 ID、PickCode、mtime、大小或 115 SHA1 等远端元数据缺失时，会优先按 `file_id` 补齐文件详情后再保存 `SyncFile` 和 STRM 文件。上传完成、远端已存在等非 Webhook 文件任务在 STRM 新增或更新后会优先提交 Emby item 级定向刷新，定位不到可靠 item 时回退同步目录关联媒体库刷新。Webhook 文件任务只有 `refresh_emby=true` 且 STRM 变更或新增元数据下载任务时才提交刷新；批量和目录扫描会等所有子任务完成或失败后统一提交目标集合。

外部程序触发 STRM 生成的接口见 [STRM Webhook](strm-webhook.md)。本文件只说明上传完成、远端已存在和 Webhook 入队后共用的 worker 后处理边界。

目录监控规则 `upload_metadata=true` 时，元数据文件上传完成后也会进入同一 STRM 生成队列。worker 不会为元数据生成 `.strm`，而是把目录监控源文件复制到同步目录的 STRM 本地路径，文件名和扩展名保持不变，并保存对应 `SyncFile`。复制前会确认上传任务来源是 `directory_monitor`，并校验当前源文件大小和 mtime 仍与上传任务记录一致；源文件不存在、已被替换或写入 STRM 本地路径失败时，STRM 任务会失败，后续源文件清理不会触发。复制发生在源文件清理之前，因此开启 `delete_source_after_success` 时不会因为先删除源文件导致元数据丢失。

“同步目录生效 STRM 配置”INFO 只在完整 STRM 同步启动时输出。上传完成后的单文件后处理仍沿用当前同步目录配置，但不会为每个后处理任务重复输出配置日志。

如果同一远端文件发生移动或重命名，服务会用 `file_id` / `pick_code` 查找旧 `SyncFile`。新 STRM 写入成功后，只精确删除旧记录里的 `local_file_path`，不会按文件名模糊删除其他 `latest` 或同名文件。旧 STRM 删除失败时，新 STRM 会保留，任务记录失败原因，便于从 `strm_generation_tasks.last_error` 和应用日志排查。

目录监控规则的 `delete_source_after_success` 默认关闭。开启后，也必须同时满足以下条件才会删除本地源文件：

- 上传任务来源为 `directory_monitor`。
- 上传任务在创建时已因规则开启删除源文件而标记为 `source_cleanup_status=pending`。
- 上传任务状态为 `completed`。
- 上传结果为 `rapid_upload`、`multipart_uploaded` 或已确认签名的 `remote_exists`。
- 关联 `StrmGenerationTask` 状态为 `completed`。
- 源文件路径仍在规则 `monitor_path` 内。
- 当前路径上的文件大小和 mtime 仍与上传任务记录一致；如果同一路径已被新文件替换，则跳过删除并记录清理失败。

删除源文件成功后，程序会从源文件所在目录向上删除空目录，但不会删除 `monitor_path` 根目录。清理失败只记录到上传任务的 `source_cleanup_status` 和 `source_cleanup_error`，不会回滚远端文件或已生成的 STRM。
