# 配置和密钥

## 默认端口

- Web 默认端口：HTTP `12333`，HTTPS `12332`
- Emby 代理默认端口：HTTP `8095`，HTTPS `8094`

管理员用户名和密码在创建和修改时使用严格校验规则：用户名去除首尾空白后长度必须为 3 到 20 个字符且只能包含英文和数字，密码长度至少 6 个字符，且不能是纯数字或纯字母。登录时校验用户名和密码非空，用户名 20 个字符上限。管理员密码使用 bcrypt 哈希保存，新生成和修改后的密码使用成本参数 `12`。旧成本哈希会在用户下一次成功登录后自动升级。

## 首次管理员初始化

`config.yaml` 不保存管理员用户名和密码。首次启动并完成数据库初始化后，如果 `users` 表为空，程序会生成一次性初始化码并写入启动日志。打开 Web 登录页后，系统会显示创建管理员表单；输入启动日志中的初始化码、管理员用户名和密码后，后端会使用 bcrypt 哈希密码并创建首个管理员。

初始化码只在用户表为空时生成，创建管理员成功后立即失效。该启动日志属于首次设置必需提示，即使 `log.level` 配置为 `error` 也会写入。`users` 表通过唯一约束保证系统只存在一个登录用户；若尚未创建管理员就重启程序，会生成新的初始化码。首次初始化应在可信网络内完成，避免无关人员访问 Web 页面。

## 浏览器登录会话

- 浏览器登录使用 `auth_token` HttpOnly Cookie，不在 Web Storage 保存 JWT。
- 服务端通过 `user_sessions` 表控制会话有效性，退出登录、修改密码、两步验证变更和登录设备撤销都会更新该表。
- `csrf_token` Cookie 可被前端读取，前端会在 `POST`、`PUT`、`PATCH`、`DELETE` 请求中通过 `X-CSRF-Token` 发送，服务端同时校验请求来源和 session 中的 CSRF 哈希。
- CORS 和 CSRF 共享可信来源判断：同源请求自动允许，默认允许 Vite 开发来源 `http://localhost:5173`、`http://127.0.0.1:5173` 和 `http://[::1]:5173`。自定义前后端跨源部署时，在 `config/config.yaml` 中配置精确来源：

  ```yaml
  trustedOrigins:
    - https://qms.example.com
  ```

  `trustedOrigins` 按 `scheme://host[:port]` 精确匹配，显式默认端口 `http:80`、`https:443` 会按无端口来源处理。前端和 API 使用同一个域名访问时不需要配置；旧配置缺少该字段会按空列表处理。
  通过 Nginx / Caddy 等反向代理绑定域名时，应保留原始 `Host` 并传递 `X-Forwarded-Proto`，这样同源判断可以按用户访问的域名生效。
- API Key 调用支持 `X-API-Key` header 和 `?api_key=` 查询参数，不需要 CSRF。`/emby/webhook` 配置默认启用鉴权，优先读取 `X-API-Key`，并保留 `?api_key=` 兼容只能配置 URL 的 Emby Webhook 场景。
- 在「系统设置 - API Key」创建密钥后，后端会生成 `qms_` 前缀加 24 位随机字符的完整密钥。完整密钥只在创建响应中返回一次；数据库 `api_keys` 表保存 `user_id`、`name`、`key_hash`、前 8 位 `key_prefix`、`is_active`、时间戳和 `last_used_at`，不保存完整明文。校验时对请求中的 Key 再做同样 SHA256，用 `key_hash` + `is_active=true` 查表。
- 本地下载反代 `/proxy-115` 仅允许访问 115 CDN 和百度网盘下载域名，用于 115 / 百度网盘播放代理和媒体信息提取；初始目标和每次重定向目标都会执行同一白名单校验，其他目标地址会被拒绝。

## SSE 反向代理

前端与 API 应通过同一 origin 访问。生产环境由 QMediaSync 托管前端；开发环境由 Vite 将相对 `/api` 代理到后端。`/api/events/stream`、`/api/logs/stream` 和 `/api/sync/tasks/:id/stream` 是持续 SSE 响应，反向代理必须关闭缓冲并使用足够长的读取超时。

Nginx 可在对应 location 设置 `proxy_http_version 1.1`、`proxy_buffering off`、`proxy_cache off`、`gzip off` 和较长的 `proxy_read_timeout`；Caddy 可使用 `flush_interval -1`。不要把 SSE 作为跨 origin Cookie 通道配置，本项目首期不提供该场景的专项支持。

## 数据库

首次启动且不存在 `config/config.yaml` 时，后端会先启动配置向导。向导当前提供 SQLite 和外部 PostgreSQL 两种选择；保存后会生成 `config/config.yaml`，旧版 `config.yml` 仍可读取。

完整配置项示例见 [config.yaml](examples/config.yaml)。示例文件只用于说明字段含义，运行时仍以 `config/config.yaml` 为准。

代码默认配置是 `postgres + embedded`。Docker 镜像会安装 `postgresql15`，可以直接配合内嵌 PostgreSQL 使用；裸二进制和本地开发环境不随仓库携带 PostgreSQL 二进制，如果要使用 PostgreSQL，建议安装 PostgreSQL 15 及以上并配置为外部数据库，或自行保证内嵌模式所需的 PostgreSQL 命令可用。

数据库引擎、配置项、迁移和维护入口的完整说明见 [数据库](database.md)。

## 115 接口监控统计

首页“115 接口监控”卡片中的请求数、QPS、QPM、QPH、平均响应时间和限流次数来自 `request_stats` 表，会在程序重启后继续按时间窗口聚合展示。

“限流中 / 运行正常”、本次等待时间、已过时间和剩余限流时间来自当前进程的 115 请求队列限流管理器。当前限流暂停时长为 1 分钟，程序重启后该运行态会恢复为未限流。

## 115 上传协议边界

115 Open API 上传分为 115 调度层和 OSS 数据层。`/open/upload/init` 负责上传初始化、秒传判断和二次认证调度；`status = 2` 表示秒传成功，`status = 1` 表示需要进入 OSS 上传，返回的 `callback` / `callback_var` 必须在 OSS 完成 multipart 时带回，让 115 完成文件入库。官方返回中的 `callback` 可能是单个对象，也可能是对象数组，后端会取第一个回调配置用于 OSS complete。传给 OSS 时，后端只按 OSS SDK V2 要求把 115 返回的两个 JSON 字符串原样 Base64 编码到 `x-oss-callback` / `x-oss-callback-var`，不本地展开 `callbackBody` 占位符，也不向 `callback_var` 注入 `bucket`、`object`、`size`、`sha1` 等非 `x:` 字段。`sha1` 占位符由 OSS 在 `CompleteMultipartUpload` 后基于最终对象生成并回填；本地 SHA1 只用于 115 `fileid`、秒传 / 续传调度和 session 校验，不作为 OSS multipart 输入。`/open/upload/get_token` 只负责获取 OSS 临时凭证，凭证不得写入数据库，也不得写入普通日志。

当前 115 Open API 非秒传上传使用 OSS multipart：先带 `sequential=1` 调用 `InitiateMultipartUpload`，按 part 上传，再 `CompleteMultipartUpload` 并带回 115 callback。该链路已通过真实非秒传任务验证；成功时 OSS 返回最终对象 SHA1，115 callback 返回 `state=true`、远端 `file_id` 和 PickCode。默认 part size 为 `32 MiB`；当文件按 `32 MiB` 切分会超过 `9999` 个 part 时，part size 会按 `ceil(file_size / 9999)` 放大并向上取整到 `1 MiB`。如果计算出的 part size 超过 OSS 单 part 上限，上传会失败并返回错误。单个 part 上传失败时至少重试 3 次；重试前会刷新 OSS 上传凭证并继续当前 part。

断点续传必须同时满足两层条件：先调用 115 `/open/upload/resume` 恢复 `file_size`、`target`、`fileid`、`pick_code` 对应的上传调度信息，再用 OSS `upload_id` 和 `ListParts` 查询已上传分片。上传队列会把 `upload_id`、part size、已上传字节数和分片进度保存到 `upload_sessions`，进程重启时只把 `uploading` 任务恢复为 `pending`，不删除 session。重试时如果本地文件大小、mtime、SHA1 和快速签名仍匹配，会优先按 session 续传；如果本地文件签名变化，会把旧 session 标记为 `aborted` 并让任务失败，避免同一路径不同文件误用旧 checkpoint。如果 OSS 明确返回 `NoSuchUpload`、`InvalidUploadId` 等 checkpoint 已失效错误，系统会把当前 session 标记为 `session_expired_restarted`，清空旧 `upload_id` 和分片进度，并在同一次任务中复用当前 115 调度结果重新创建 OSS multipart。仅重新调用 init、仅普通 multipart 上传，或发现远端已有同名同 SHA1 / 同大小文件，都不能称为断点续传。

上传前会检查远端同路径文件；只有同名目标的 SHA1 和大小都与本地文件一致时，才跳过真实上传并把任务结果记为 `remote_exists`。这类任务仍会写入完成后的远端文件 ID / PickCode，并按需创建 STRM 生成任务；它不是断点续传。

`preid` 按 115 官方“文件上传”文档使用文件前 `128 KiB` 的 SHA1。本项目以官方文档为准，并把该窗口封装为可测试实现，避免上传流程中散落协议常量。

秒传等待策略保存于 `settings` 表，默认关闭。启用后，上传初始化未命中秒传时会按 `upload_rapid_wait_interval_seconds` 间隔重复 init，最长等待 `upload_rapid_wait_timeout_seconds`；`upload_rapid_wait_interval_seconds` 只控制重试频率，`upload_rapid_wait_timeout_seconds` 才是最大等待上限，最后一次等待会按剩余超时时间裁剪。`upload_rapid_wait_min_size` 和 `upload_rapid_wait_force_size` 用于限制哪些文件进入等待策略，`upload_rapid_wait_skip_upload` 用于控制等待超时后是否跳过真实上传。

115 直链缓存有效性检查保存于 `settings` 表，默认开启，默认总超时为 3 秒，可配置范围为 1 到 9 秒。该配置只影响 115 直链缓存命中后的 HEAD 检查；关闭后会直接使用缓存链接，不再对缓存 URL 发起 HEAD 请求。为避免同一缓存键并发请求长时间等待，服务端运行时会将旧配置中的更大超时值裁剪到 9 秒。百度网盘和 OpenList 不使用这套有效性检查机制。

自动化测试覆盖协议字段、part size、`sequential=1`、callback 校验和本地 mock 请求。真实 115 小文件 / 大文件上传实测需要有效 115 Open API 沙箱账号和可写测试目录，且会产生远端写入副作用；未获得明确沙箱授权前，不应在开发环境自动执行真实外部上传。排查非秒传失败时，`115.log` 会记录 115 Open API 返回、OSS multipart 初始化、ListParts、UploadPart、CompleteMultipartUpload 请求边界和 callback 业务返回；日志会避免明文输出 STS 密钥和 SecurityToken。

## 目录监控上传

同步目录基础配置与目录监控上传规则通过 `POST /api/sync/paths` 或 `PUT /api/sync/paths/:id` 原子保存；旧 `/api/sync/path-add`、`/api/sync/path-update` 和规则独立写接口已移除。更新同步目录时，请求中的 `source_type` 和 `account_id` 必须与旧记录一致。请求中的 `directory_upload.enabled` 是同步目录总开关，规则内的 `enabled` 是单条规则开关；程序启动和重载时只加载两者都开启的规则。关闭总开关不会修改规则自身 `enabled`，下次开启时会恢复上一次的规则启停状态。创建请求可携带 `Idempotency-Key`，同一 key 已完成时回放已创建聚合，不会重复创建 SyncPath；该 key 仍处于处理中时返回 `IDEMPOTENCY_CONFLICT`。规则的 `remote_root_path` 和 `remote_root_id` 都不能为空，目标目录必须位于该同步目录的远端路径下。`watch_mode=auto` 会根据运行环境选择 fsnotify 或 polling；显式 `fsnotify` 初始化失败时规则启动失败，`polling` 使用内置 30 秒间隔查漏。`recursive=false` 时只处理监控根目录文件。

查漏扫描和 fsnotify 事件默认只把候选视频文件加入稳定性队列；规则 `upload_metadata=true` 时，也会把当前同步目录识别为元数据扩展名的文件加入队列。视频扩展名和元数据扩展名都沿用同步目录配置：同步目录启用自定义配置且数组非空时使用自定义值，否则回退全局 STRM 设置。稳定性检查间隔、稳定窗口和补偿扫描间隔均为代码内置：稳定性检查每 2 秒执行一次，文件需要在 15 秒内保持 `v1:size:mtime_ns` fingerprint 不变，并连续 3 次检查不变；补偿扫描每 30 秒执行一次。启动查漏、polling 定期查漏和 fsnotify 新目录补偿扫描会提交到同一个内置扫描执行器；执行器按 `rule_id + clean(root)` 合并已排队或执行中的重复目录，已取消的同 key 扫描会允许后续请求重新提交，默认同时最多执行 2 个扫描任务，不提供页面或接口配置。polling 模式会在运行时维护 `relative_path -> source_fingerprint` 快照，每轮只把新增或 fingerprint 变化的文件加入稳定性队列；`startup_scan_enabled=true` 时，启动查漏会处理已有文件并初始化快照，避免第一轮 polling 重复提交同一 fingerprint；`startup_scan_enabled=false` 时，启动时会先建立 baseline 快照，不处理已有文件，之后只处理新增或变化文件；baseline 建立遇到非取消类扫描错误时会记录日志并由后续 polling 重试，已成功扫描到的部分仍作为 baseline。polling 定期扫描遇到非取消类中途错误时，不会用本轮 partial snapshot 替换完整快照，避免误删已知 fingerprint；但本轮已成功扫描部分中新增或变化的文件仍会加入稳定性队列，下一轮继续重试。启动查漏提交前会同步校验同步目录、监控路径和扫描根目录，基础校验失败会阻止规则启动；实际扫描期间发生的错误会写入应用日志，不阻塞已经启动的规则。fsnotify 文件事件候选在通过递归、忽略规则和扩展名过滤后，会先按 `rule_id + relative_path + source_fingerprint` 做 recently queued 内存 TTL 去重，再进入稳定性队列；`source_fingerprint` 使用同样的 `v1:size:mtime_ns`。这个缓存只用于减少同一 fsnotify 事件风暴造成的重复入队，不写入 `directory_upload_processed_files`，也不作用于启动查漏、手动扫描或 polling 补偿扫描。默认忽略隐藏路径、带 `.part`、`.tmp`、`.download`、`.aria2`、`.torrent` 临时后缀的文件或目录，以及 `@Recycle`、`#recycle`、`.Trash`、`.Trashes` 回收站目录和规则中的 `ignore_patterns`；不会按 `.nfo`、`.jpg`、`.png` 等元数据扩展名做默认忽略，`upload_metadata=true` 时仍可按同步目录配置上传元数据文件。规则查询接口会把忽略规则以 `ignore_patterns` 数组返回。目录监控上传不在事件 goroutine 内直接上传文件；如果确认远端目录、检查远端同名文件或写入上传任务等入队前步骤发生临时错误，文件会重新进入稳定性队列等待下一轮处理，已经创建成功的上传任务仍由全局上传队列按既有重试上限处理。

创建上传任务前会检查 115 同目录同名文件。远端文件大小和 SHA1 都与本地一致时直接标记为远端已存在并创建 STRM；大小或 SHA1 不一致时按 `overwrite_mode` 处理：`skip_same` 跳过、`fail_conflict` 停止、`replace_conflict` 删除远端旧文件后重新上传。

统一保存接口中的 `rules` 是最终完整规则集合，已有规则未出现在请求中即视为删除；总开关开启时至少需要一条规则自身启用，总开关关闭时允许空规则集合。删除同步目录时，会在同一删除事务内清理其目录监控上传规则，并在接口成功后重载目录监控上传服务，避免已删除同步目录对应的 watcher 继续运行。

`delete_source_after_success` 默认关闭。开启后，只有任务创建时已标记为待清理的目录监控上传任务，才会在上传成功且关联 STRM 生成任务成功后删除本地源文件。删除前再次校验 fingerprint 与路径边界。清理依赖以 `strm_generation_tasks.upload_task_id` 关联上传任务为准；清理由 worker 完成事件和服务启动/周期补偿扫描两类幂等路径触发。删除规则、关闭删除开关或修改监控边界时，会先把相关 pending cleanup 改为 `none` 并记录取消原因，再删除 processed 账本，不追溯删除本地源文件。

`upload_metadata=true` 时，目录监控上传成功的元数据文件会在 STRM 生成 worker 中复制一份到同步目录的 STRM 本地路径。复制使用同步目录的远端路径映射规则，保留原文件名和扩展名，不生成 `.strm`。复制发生在源文件清理之前；如果复制前发现源文件不存在、fingerprint 已变化，或者写入目标路径失败，STRM 任务会失败并阻止后续源文件删除。

完整流程见 [上传和 STRM 后处理流程](upload-strm-workflow.md)。

## STRM 生成后处理

115 上传任务成功或确认远端同名同 SHA1 / 同大小文件已存在后，会按远端文件 ID / PickCode 创建 `strm_generation_tasks`。后台 STRM 生成 worker 会读取待处理任务，并复用现有同步目录配置写入 STRM 内容，因此仍兼容 `strm_base_url`、`add_path`、PickCode 和账号用户 ID 等既有 STRM 规则。该后处理不会创建 `syncs` 同步记录，也不会把同步目录状态改成等待中；完整同步记录只由手动同步、定时同步等 STRM 同步入口创建。

文件级任务只传 `file_id`，或传入了 `file_id` 但缺少文件名、路径、父目录 ID、PickCode、mtime、大小或 115 SHA1 等远端元数据时，服务会通过对应网盘 driver 补齐文件详情。写入流程会先确认新 STRM 内容需要新增或更新；确认需要更新后会直接写入新 STRM，不会再次执行内容一致性比较，因此同一次后处理只会输出一次 PickCode、路径或用户 ID 差异日志。如果同一 `file_id` / `pick_code` 的远端路径变化，会在新 STRM 写入成功后精确删除旧 `SyncFile.LocalFilePath` 对应的旧 STRM。旧路径清理失败不会删除新 STRM，也不会把任务标记完成，任务会记录错误并保留为失败状态供后续排查或重试。目录监控上传的元数据文件会在同一 worker 中复制到 STRM 本地路径并保存 `SyncFile`，但不会提交 STRM 内容刷新。

115 同一目录下的视频如果去掉扩展名后映射到同一个本地 STRM，例如 `episode.mkv` 和 `episode.mp4` 都映射为 `episode.strm`，系统只允许上传时间 `Ptime` 最新的文件生成和更新该 STRM；上传时间相同时使用 FileID 固定排序，避免结果受分页顺序或并发调度影响。完整同步会先完成远端文件和路径收集，再在内存中选择 owner，不增加 115 文件列表请求；目录扫描会在已有目录列表中直接过滤 non-owner。上传完成、远端已存在和 Webhook 等单文件任务优先查询本地 `SyncFile`，只有发现同目标路径候选时才额外请求一次远端父目录确认 owner。相同目标路径的比较和写入使用进程内路径锁串行执行。

“同步目录生效 STRM 配置”INFO 只在完整 STRM 同步启动时输出。上传完成后的单文件 STRM 后处理仍使用相同的生效配置，但不会为每个后处理任务重复输出该配置日志。

上传完成、远端已存在等非 Webhook STRM 任务在 STRM 新增或更新后会优先提交 Emby item 级定向刷新：已有 Movie、Video、Episode 关联时刷新对应 item；同季新增剧集没有自身 Episode 关联时优先刷新已有 Season，缺少 Season 时刷新 Series。无法从本地 Emby 索引、PickCode 或路径兜底查询定位可靠 item 时，自动回退到同步目录关联媒体库刷新。刷新仍进入 `emby_library_refresh_tasks` 协调器去抖和等待队列，不在 STRM 生成流程内直接请求 Emby。Webhook 请求默认不刷新，只有 `refresh_emby=true` 且 STRM 变更或新增元数据下载任务时才提交刷新；批量和目录扫描只有在所有子任务成功完成且存在 STRM / 元数据变化时才统一提交，任一子任务失败则父任务失败且不提交刷新。STRM 内容无变化时只更新 / 确认 `SyncFile`，不重复提交刷新任务。

外部程序可以通过 [STRM Webhook](strm-webhook.md) 创建同一类 STRM 生成任务，接口字段、响应格式和幂等边界以独立文档为准。

## Emby 302 出站 HTTPS

Emby 302 代理请求 Emby、OpenList、m3u8 和下载资源时，默认启用 HTTPS 证书校验。共享 HTTP client 会复用空闲连接，避免同一上游的连续请求反复建立 TCP / TLS 连接。

只有在受控内网自签名证书或临时排障场景下，才建议显式跳过证书校验：

```yaml
emby302:
  insecure_skip_verify: false
```

将 `emby302.insecure_skip_verify` 改为 `true` 后，Emby 302 出站 HTTPS 请求会接受无法验证的证书。程序启动时会写入风险提示日志；该模式存在中间人攻击风险，不建议在公网或长期生产环境开启。

## 日志行为和脱敏

日志文件路径由 `config/config.yaml` 的 `log` 配置决定，默认相对于配置目录写入：

| 配置项 | 默认文件 | 用途 |
| --- | --- | --- |
| `log.level` | `info` | 日志等级，可选值：`debug`、`info`、`warn`、`error`。 |
| `log.maxSizeMB` | `10` | 单个轮转日志文件最大大小，单位 MB，范围 `1-1024`。 |
| `log.maxBackups` | `3` | 每个日志文件最多保留的轮转备份数，范围 `1-100`。 |
| `log.maxAgeDays` | `7` | 轮转备份最长保留天数，范围 `1-365`。 |
| `log.app` | `logs/app.log` | 主程序日志，包含 Web、控制器、模型、Emby 302 等通用日志。 |
| `log.v115` | `logs/115.log` | 115 开放平台相关请求和队列日志。 |
| `log.openList` | `logs/openList.log` | OpenList 相关日志。 |
| `log.tmdb` | `logs/tmdb.log` | TMDB 刮削相关日志。 |
| `log.baiduPan` | `logs/baidupan.log` | 百度网盘相关日志。 |
| `log.web` | `logs/web.log` | 预留 Web 日志配置。 |
| `log.syncLogDir` | `logs/sync` | 同步任务独立日志目录配置。 |

历史配置项 `log.file` 仍可读取；当 `log.app` 为空且 `log.file` 有值时，程序会把 `log.file` 作为主程序日志路径使用。新保存的配置统一写入 `log.app`。

同步任务日志文件名为 `sync_<任务 ID>.log`，默认写入 `logs/sync`。旧版本写入的 `logs/libs/sync_<任务 ID>.log` 仍可在任务详情和日志接口中读取；新启动不再主动创建空的 `logs/libs` 目录。

全局日志默认参与轮转，包括主程序日志、115 日志、OpenList 日志、TMDB 日志和百度网盘日志。轮转由写入触发：当当前日志文件达到 `log.maxSizeMB` 后，`lumberjack` 会把旧文件改名为备份文件，并继续写入新的当前日志文件。旧日志固定启用压缩。`log.maxBackups` 表示每个日志最多保留多少个轮转备份，`log.maxAgeDays` 表示备份文件最长保留多少天，两者任一条件达到后旧备份都可能被清理。

同步任务日志不参与轮转。每个同步任务写入独立的 `sync_<任务 ID>.log`，并随着同步记录清理一起删除。定时任务每天清理创建时间早于 7 天的同步记录和对应同步日志。

当前自定义 `QLogger` 支持运行时日志等级过滤，日志格式保持为 `YYYY/MM/DD HH:MM:SS.micro [LEVEL] message`。默认 `info` 会写入 `Info`、`Warn`、`Error`，不写入 `Debug`；`debug` 会写入全部等级；`warn` 只写入 `Warn` 和 `Error`；`error` 只写入 `Error`。`Fatalf`、`Panicf` 和运行必需的启动警告始终输出；首次管理员初始化码属于运行必需的启动警告。`gin.ReleaseMode` 只影响 Gin 自身模式，不控制 `QLogger` 的输出。

可以在 Web 页面「系统设置 - 日志设置」调整日志等级和全局日志轮转参数，后端会保存到 `config/config.yaml` 并立即更新运行中的全局日志器，无需重启。日志查看页和同步任务详情页的日志等级筛选只影响当前页面显示，不影响日志文件写入。

运行日志默认会在写入前完全脱敏常见敏感字段，包括 `api_key`、`X-Emby-Token`、`Authorization`、`X-Emby-Authorization`、`X-API-Key`、`password`、`access_token`、`refresh_token`、`AccessKeySecret`、`SecurityToken`、`Cookie` 等。普通 `Info`、`Warn`、`Error` 和 `Debug` 日志都会执行脱敏，不保留敏感值开头或结尾字符。脱敏后的敏感值统一显示为 `******`。Emby 302 的 `Tip`、`Progress` 等日志也会写入同一 `QLogger` 管道；请求头保持 `名称=值` 的排查格式，敏感值脱敏后非敏感字段仍会保留。

Emby 302 的嵌入配置默认设置 `log.disable-color: true`，避免 ANSI 彩色控制字符进入控制台重定向、日志文件或日志采集系统。若仅在支持颜色的交互式终端中查看，可以在 `backend/emby302.yaml` 中改为 `false` 后重新构建。

需要临时排查 Emby 302 等链路的完整请求信息时，可以在本地调试环境设置 `QMS_UNSAFE_SENSITIVE_LOG=1`。该开关只影响显式标记为敏感的 `SensitiveDebug` 日志；启用后这类 Debug 日志可能包含 API Key、Token、Cookie 或密码，程序启动时会写入风险提示。不应在生产环境长期打开，也不应分享对应日志文件。

## 需要自备的密钥

- 115 开放平台 APP ID：前端支持扫码授权和网页授权；自定义 APP ID 走扫码授权。
- TMDB API Key / Access Token：可在 Web 页面「刮削设置」填写；留空时使用默认值。
- OpenAI 兼容 API Key：默认对接硅基流动（SiliconFlow），可在 Web 页面「刮削设置」填写。
- fanart.tv API Key：可在 Web 页面「刮削设置」填写。

以上默认密钥可在 `backend/main.go` 开头的变量中设置、编译时通过 ldflags 传入，或运行时通过环境变量 / `config/.env` 注入（变量名 `TMDB_API_KEY`、`TMDB_ACCESS_TOKEN`、`SC_API_KEY`、`FANART_API_KEY`）。

取值优先级：Web UI > 环境变量 / `config/.env` > ldflags。`config/.env` 会覆盖真实环境变量。

## 本地敏感数据

两步验证等本机敏感数据使用实例本地密钥：每个实例首次启动自动生成并保存到 `config/encryption.key`。

`jwtSecret` 是 JWT Cookie 会话票据签名密钥。启动时如果为空、仍为当前公开默认值，或仍为历史版本的公开默认值，程序会自动生成 32 字节强随机密钥并写回 `config/config.yaml`；如果配置目录不可写，启动会失败。

修改 `jwtSecret` 会让现有登录 Cookie 无法通过签名校验，用户需要重新登录。

网盘 OAuth 中转使用共享密钥 `OAUTH_RELAY_ENCRYPTION_KEY`，可编译时通过 ldflags 变量 `main.OAuthRelayEncryptionKey` 传入，或运行时通过环境变量 / `config/.env` 注入。
