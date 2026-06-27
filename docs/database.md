# 数据库

## 总览

QMediaSync 当前支持 `SQLite` 和 `PostgreSQL` 两种数据库引擎。默认配置是 `postgres + embedded`，也就是由程序启动内嵌 PostgreSQL 进程。

数据库结构、默认数据和版本迁移都统一由 `backend/internal/models/migrator.go` 处理，这个文件是数据库侧的单一事实来源：

- `AllTables` 定义了建表、修复、备份和恢复会遍历的表集合。
- `InitDB()` 负责首次启动时的全量初始化。
- `Migrate()` 负责已有数据库的版本升级。
- `BatchCreateTable()` 负责补齐缺失的表、字段和索引。
- `BatchRepairTableSeq()` 负责 PostgreSQL 的主键序列修复。

说明：

- 除 `internal/notification` 里的通知表外，项目大多数表都嵌入 `BaseModel`，主键和时间字段都是 `int64` Unix 秒。
- `internal/notification` 里的通知表使用 `time.Time` 作为 `CreatedAt` / `UpdatedAt`。
- 带 `gorm:"-"` 的字段不入库，只用于运行时组装、前端回传或关联对象。
- PostgreSQL 数据库名会在首次配置、外部库自动创建和内嵌库初始化时统一校验并作为 identifier 引用；名称不能为空，不能超过 63 字节，不能包含空白或控制字符。

## 迁移与初始化

### 首次启动

当 `migrator` 表不存在时，`InitDB()` 会直接执行：

1. `BatchCreateTable()`：对 `AllTables` 逐表执行 `AutoMigrate`。
2. `InitMigrationTable(MaxVersionCode)`：写入当前版本号，当前值是 `46`。
3. `InitSettings()`：创建默认 `settings` 记录。
4. `InitUser()`：创建默认管理员用户。
5. `InitScrapeSetting()`：创建默认刮削配置和默认分类。
6. `InitEmbyConfig()`：创建默认 `emby_config` 记录。

这意味着空库首次启动时，不会逐个版本回放历史迁移，而是直接初始化到当前结构版本。

### 已有数据库

当 `migrator` 表已存在时，`Migrate()` 会读取 `version_code`，然后按顺序执行版本补丁。每一步更新后都会把版本号加一，因此一次启动可以连续跨过多个历史版本。

### 关键版本

| 版本 | 变更 |
| --- | --- |
| 35 | 拆分 `emby_config`，并清理重复的 `scrape_settings` 记录。 |
| 36 | `settings` 新增 `file_list_page_size`。 |
| 37 | `emby_config` 新增播放剧情简介和播放进度开关。 |
| 38 | 为已有通知渠道补齐 `playback_*` 和 `scrape_error` 规则。 |
| 39 | `account` 新增 `app_id_name`。 |
| 40 | `account` 新增 `auth_source_type` 和 `auth_provider`。 |
| 41 | `users`、`db_download_tasks`、`db_upload_tasks` 补齐两步验证和队列重试字段。 |
| 42 | 新增 `emby_library_refresh_tasks`。 |
| 43 | 下载 / 上传任务的 `source` 从展示文案迁移为稳定存储值。 |
| 44 | 任务来源枚举迁移后的结构版本。 |
| 45 | 新增 `user_sessions` 表，用于浏览器登录会话撤销、CSRF 校验和登录设备管理。 |
| 46 | 当前数据库版本；通知渠道类型索引从唯一索引改为普通索引，并为已有渠道补齐缺失通知规则。 |

## 修复与重建

后台的数据库修复接口是 `POST /api/database/repair`，对应 `RepairDB()`：

1. 调用 `BatchCreateTable()`。
2. 调用 `BatchRepairTableSeq()`。

对 `AllTables` 做一次 `AutoMigrate`：

- 缺失的表会被创建。
- 缺失的列和索引会被补齐。
- 已有数据不会被主动删除。
- PostgreSQL 下会顺手把每张表的 `id` 序列重置到当前最大值。

真正的清库操作是 `POST /api/database/delete-all-table`，它会执行 `BatchDropTable()`，把 `AllTables` 里的表全部删除，风险很高。

## 常用枚举

以下字段存的是稳定值，不是展示文案：

| 字段 | 存储值 |
| --- | --- |
| `source_type` | `115`、`local`、`123`、`openlist`、`baidupan`、`emby_media` |
| `media_type` | `movie`、`tvshow`、`other` |
| `scrape_type` | `only_scrape`、`scrape_and_rename`、`only_rename` |
| `rename_type` | `hard_symlink`、`soft_symlink`、`move`、`copy` |
| `enable_ai` | `off`、`assist`、`enforce` |
| `sync.status` | `0` 待处理、`1` 进行中、`2` 已完成、`3` 失败 |
| `sync.sub_status` | `0` 无、`1` 正在处理网盘文件、`2` 正在处理本地文件列表 |
| `scrape_media.status` | `unscanned`、`scanned`、`scraping`、`scraped`、`renaming`、`renamed`、`rename_failed`、`ignore`、`scrape_failed`、`rollbacking` |
| `download.status` | `0` 待下载、`1` 下载中、`2` 已完成、`3` 失败、`4` 已取消 |
| `download.source` | `strm_sync`、`local_file`、`emby_media` |
| `upload.status` | `0` 等待中、`1` 上传中、`2` 已完成、`3` 失败、`4` 已取消 |
| `upload.source` | `strm_sync`、`scrape_organize` |
| `backup.status` | `pending`、`running`、`completed`、`failed`、`cancelled`、`timeout` |
| `backup.type` | `manual`、`auto` |
| `notification.type` | `sync_finish`、`sync_error`、`scrape_finish`、`scrape_error`、`system_alert`、`media_added`、`media_removed`、`playback_start`、`playback_pause`、`playback_stop` |
| `notification.priority` | `high`、`normal`、`low` |

## 表结构

### `migrator`

数据库版本表，记录当前结构版本。

- `id`：固定为 `1`。
- `created_at` / `updated_at`：创建和更新时间。
- `version_code`：当前数据库版本号，当前值为 `46`。

### `users`

登录用户表。

- `username`：登录名，唯一。
- `password`：bcrypt 哈希后的密码。
- `two_factor_enabled`：是否启用两步验证。
- `two_factor_secret`：加密后的 TOTP 密钥。
- `two_factor_pending_secret`：启用确认前的临时 TOTP 密钥。

### `account`

网盘 / OpenList / 115 授权账号表。

- `name`：账号备注，便于人工识别。
- `source_type`：账号来源类型，见上面的 `source_type` 枚举。
- `app_id`：115 开放平台应用 ID。
- `app_id_name`：自定义应用显示名。
- `auth_source_type`：115 授权来源类型。
- `auth_provider`：115 授权 Provider。
- `token` / `refresh_token`：访问凭据。
- `token_expiries_time`：凭据过期时间戳。
- `user_id`：账号对应的用户 ID。
- `username`：网盘用户名或 OpenList 登录用户名。
- `password`：OpenList 登录密码。
- `base_url`：OpenList 访问地址。
- `token_failed_reason`：刷新 Token 失败原因。

补充：

- `app_id_name` 是后加字段，用于区分自定义 115 应用。
- `auth_source_type`、`auth_provider` 是后加字段，用于稳定描述授权来源。

### `api_keys`

API Key 认证表。

- `user_id`：所属用户 ID。
- `name`：Key 名称或用途说明。
- `key_hash`：API Key 的 SHA256 哈希，只存哈希，不存明文。
- `key_prefix`：前 8 位明文前缀，用于列表展示。
- `last_used_at`：最后使用时间戳。
- `is_active`：是否启用。

### `user_sessions`

浏览器登录会话表。JWT 只作为客户端 Cookie 中的会话票据，实际有效性以本表为准。

- `session_id`：会话 ID，写入 JWT claims，用于查表。
- `token_id`：JWT `jti`，用于审计和排查。
- `user_id` / `username`：关联用户和登录用户名快照。
- `csrf_token_hash`：CSRF Token 的 SHA256 哈希，原始值只保存在前端可读 `csrf_token` Cookie 和内存状态中。
- `user_agent` / `ip_address`：登录设备信息。
- `expires_at`：会话过期时间戳。
- `last_seen_at`：最后活跃时间戳。
- `revoked_at` / `revoke_reason`：撤销时间和原因。

### `settings`

全局配置表，包含线程、STRM 和历史兼容字段。

- `download_threads`：下载队列并发数。
- `file_detail_threads`：115 文件详情请求并发数。
- `openlist_qps`：OpenList QPS。
- `openlist_retry`：OpenList 重试次数。
- `openlist_retry_delay`：OpenList 重试间隔，单位秒。
- `file_list_page_size`：115 文件列表分页大小。

STRM 相关字段：

- `local_proxy`：本地代理开关，`-1` 表示使用路径自身配置。
- `strm_base_url`：生成 STRM 的基础地址。
- `cron`：定时任务表达式。
- `min_video_size`：最小视频大小，单位字节。
- `video_ext`：视频扩展名 JSON 字符串。
- `meta_ext`：元数据扩展名 JSON 字符串。
- `exclude_name`：排除文件名 JSON 字符串。
- `upload_meta`：是否上传元数据。
- `download_meta`：是否下载元数据。
- `delete_dir`：是否删除目录。
- `add_path`：是否添加路径。
- `check_meta_mtime`：是否检查元数据修改时间。

历史兼容字段：

- `use_telegram`、`telegram_bot_token`、`telegram_chat_id`、`meow_name`：旧通知配置，已迁移。
- `emby_url`、`emby_api_key`：旧 Emby 配置，已迁移。
- `http_proxy`：HTTP 代理地址。

### `sync_paths`

同步目录表，保存一个同步源和它对应的 STRM / 元数据生成规则。

`sync_paths` 复用了 `SettingStrm` 的字段，因此也包含这些列：

- `local_proxy`
- `strm_base_url`
- `cron`
- `min_video_size`
- `video_ext`
- `meta_ext`
- `exclude_name`
- `upload_meta`
- `download_meta`
- `delete_dir`
- `add_path`
- `check_meta_mtime`

额外字段：

- `custom_config`：是否使用自定义配置。
- `base_cid`：同步源目录 ID，115 / 123 需要。
- `local_path`：本地 STRM / 元数据输出目录。
- `remote_path`：同步源路径。
- `source_type`：来源类型，见枚举。
- `account_id`：对应账号 ID。
- `enable_cron`：是否启用定时同步。
- `last_sync_at`：上次同步时间戳。
- `is_full_sync`：是否全量同步。

运行时字段：

- `account_name`：账号显示名，不入库。
- `is_running`：当前运行状态，不入库。

### `sync_path_scrape_paths`

同步目录与刮削目录的关联表。

- `sync_path_id`：同步目录 ID。
- `scrape_path_id`：刮削目录 ID。

### `syncs`

同步任务记录表。

- `sync_path_id`：对应同步目录。
- `status`：任务主状态。
- `sub_status`：任务子状态。
- `file_offset`：文件偏移量，用于断点续跑。
- `total`：总文件数。
- `finish_at`：完成时间戳。
- `new_strm`：新增 STRM 数量。
- `new_meta`：新增元数据数量。
- `new_upload`：新增上传数量。
- `net_file_start_at` / `net_file_finish_at`：处理网盘文件开始 / 结束时间。
- `local_file_start_at` / `local_file_finish_at`：处理本地文件列表开始 / 结束时间。
- `local_path`：本地同步路径。
- `remote_path`：远程同步路径。
- `base_cid`：根目录 ID。
- `fail_reason`：失败原因。
- `is_full_sync`：是否全量同步。

运行时字段：

- `sync_path`：关联的同步路径对象。
- `logger`：日志句柄。

### `sync_files`

同步文件表，记录单个文件在同步链路中的状态。

- `source_type`：来源类型。
- `account_id`：所属账号。
- `sync_path_id`：所属同步目录。
- `file_id`：文件 ID。
- `parent_id`：父目录 ID。
- `file_name`：文件名。
- `file_size`：文件大小。
- `file_type`：115 文件类型。
- `pick_code`：115 PickCode。
- `sha1`：文件 SHA1。
- `mtime`：最后修改时间戳。
- `local_file_path`：本地完整文件路径。
- `path`：绝对路径，不含文件名。
- `is_video`：是否视频文件。
- `is_meta`：是否元数据文件。
- `openlist_sign`：OpenList 文件签名。
- `uploaded`：是否已上传完成。
- `thumb_url`：缩略图地址。
- `processed`：是否已处理。

运行时字段：

- `sync_path`、`sync`、`account`：关联对象，不入库。

### `scrape_settings`

全局刮削设置表。

- `tmdb_url`：TMDB API 地址。
- `tmdb_image_url`：TMDB 图片地址。
- `tmdb_api_key`：TMDB API Key。
- `tmdb_access_token`：TMDB Access Token。
- `tmdb_language`：TMDB 语言。
- `tmdb_image_language`：TMDB 图片语言。
- `tmdb_enable_proxy`：是否启用 TMDB 代理。
- `enable_ai`：AI 识别模式，`off` / `assist` / `enforce`。
- `ai_base_url`：AI 服务基础地址。
- `ai_api_key`：AI API Key。
- `ai_model_name`：AI 模型名。
- `ai_prompt`：AI 提示词。
- `ai_timeout`：AI 超时时间，单位秒。
- `fanart_api_key`：fanart.tv API Key。

说明：

- `tmdb_*` 和 `fanart_api_key` 会在运行时和 `helpers` 包里的默认值合并。
- `tmdb_enable_proxy` 打开时会使用 `settings.http_proxy` 作为通用代理。

### `scrape_paths`

刮削目录表，保存扫描源、命名规则、分类规则和定时任务信息。

基础字段：

- `account_id`：账号 ID。
- `source_type`：来源类型。
- `media_type`：媒体类型，`movie` / `tvshow` / `other`。
- `source_path`：源路径。
- `source_path_id`：源路径 ID。
- `dest_path`：目标路径。
- `dest_path_id`：目标路径 ID。
- `scrape_type`：刮削类型。
- `rename_type`：重命名类型。
- `folder_name_template`：文件夹模板。
- `file_name_template`：文件名模板。

规则字段：

- `deleted_keyword`：要删除的关键词 JSON 字符串。
- `enable_category`：是否启用分类。
- `video_ext`：视频扩展名 JSON 字符串。
- `min_video_file_size`：最小视频文件大小。
- `exclude_no_image_actor`：是否排除没有图片的演员。
- `enable_ai`：AI 识别模式。
- `ai_prompt`：AI 提示词。
- `force_delete_source_path`：是否强制删除源路径。
- `enable_fanart_tv`：是否启用 fanart.tv。
- `max_threads`：最大刮削线程数。

定时字段：

- `enable_cron`：是否启用定时任务。
- `cron_expression`：Cron 表达式。
- `cron_description`：Cron 描述。
- `last_cron_run`：上次执行时间。
- `next_cron_run`：下次执行时间。
- `cron_enabled`：定时任务启用状态。

状态字段：

- `is_scraping`：是否正在刮削。

运行时字段：

- `delete_keyword`、`video_ext_list`：数组视图，不入库。
- `v115_client`、`baidu_pan_client`、`openlist_client`：运行时客户端。
- `exists_files`、`scrape_root_path`、`category`、`category_map`、`tvshow_renamed_cache`、`episode_finish_channel`、`running`、`mutex`、`is_task_running`：运行时状态。

### `scrape_strm_paths`

刮削目录和同步目录的关联表。

- `scrape_path_id`：刮削目录 ID。
- `strm_path_id`：同步目录 ID。

### `movie_categories`

电影分类表。

- `name`：分类名称。
- `genre_ids`：TMDB genre ID 的 JSON 数组。
- `language`：语言过滤的 JSON 数组。

### `tv_show_categories`

剧集分类表。

- `name`：分类名称。
- `genre_ids`：TMDB genre ID 的 JSON 数组。
- `countries`：国家过滤的 JSON 数组。

### `scrape_path_categories`

刮削目录与分类的映射表。

- `scrape_path_id`：刮削目录 ID。
- `category_id`：分类 ID。
- `file_id`：分类对应的文件 ID。

说明：

- 115 场景下是文件 ID。
- 123 场景下是文件夹 ID。
- 本地和 OpenList 场景下是相对路径。

### `scrape_media_files`

待刮削媒体表，也是刮削流程中最复杂的一张表。

关联与来源：

- `scrape_path_id`：所属刮削目录。
- `media_type`、`source_type`、`scrape_type`、`rename_type`：当前文件的业务类型。
- `enable_category`：是否启用二级分类。
- `source_path`、`source_path_id`、`dest_path`、`dest_path_id`：来源和目标路径信息。
- `media_id`、`media_season_id`、`media_episode_id`：关联的媒体、季、集记录。

识别与文件信息：

- `name`、`year`、`tmdb_id`：基础媒体匹配信息。
- `season_number`、`episode_number`：季 / 集编号。
- `path`、`path_id`：媒体文件夹路径和路径 ID。
- `tvshow_path`、`tvshow_path_id`：剧集根路径和路径 ID。
- `video_filename`、`video_file_id`、`video_pick_code`：视频文件位置和识别码。
- `nfo_path`、`nfo_file_name`、`nfo_file_id`、`nfo_pick_code`：NFO 文件信息。
- `image_files_json`、`subtitle_file_json`：图片 / 字幕文件列表 JSON。
- `tvshow_files_json`、`season_files_json`：剧集和季文件列表 JSON。

媒体分析结果：

- `resolution`、`resolution_level`、`is_hdr`：分辨率和 HDR 信息。
- `video_codec_json`、`audio_codec_json`、`subtitle_codec_json`：ffprobe 结果。

整理结果：

- `status`：当前处理状态。
- `failed_reason`：失败原因。
- `scan_time`、`scrape_time`、`rename_time`、`re_scrape_time`：各阶段时间戳。
- `category_name`、`scrape_path_category_id`：分类结果。
- `new_path_name`、`new_season_path_name`、`new_path_id`、`new_season_path_id`：整理后的新路径。
- `new_video_base_name`：整理后的视频文件名（不含扩展名）。
- `video_ext`：视频扩展名。
- `is_re_scrape`：是否重新刮削。
- `batch_no`：同一次扫描的批次号。
- `tv_is_rename`、`season_is_rename`：剧集 / 季重命名状态。

运行时字段：

- `tvshow_files`、`season_files`、`video_codec`、`audio_codec`、`subtitle_codec`、`media`、`media_season`、`media_episode`、`scrape_root_path`：运行时对象，不入库。
- `v115_client`、`baidu_pan_client`、`openlist_client`、`exists_files`、`category`、`category_map`、`tvshow_renamed_cache`、`episode_finish_channel`、`running`、`mutex`、`is_task_running`：运行时状态，不入库。

### `media`

刮削后的媒体主表，电影和剧集都对应一条记录。

基础识别字段：

- `scrape_path_id`：来源刮削目录。
- `tmdb_id`、`imdb_id`、`name`、`year`、`original_name`：媒体基础标识。
- `media_type`：电影、剧集或其他。
- `release_date`：上映或首播时间。
- `original_language`：原始语言。
- `original_country_json`、`genres_json`：国家和流派的 JSON 字符串。
- `runtime`：时长，单位分钟。
- `last_air_date`、`number_of_episodes`、`number_of_seasons`：剧集信息。
- `num`：番号。
- `mpaa_rating`：分级信息。

画面资源：

- `poster_path`：竖版海报。
- `backdrop_path`：背景图。
- `logo_path`：Logo。
- `thumb_path`、`landscape_path`、`banner_path`：其他备用封面。

正文信息：

- `overview`：简介。
- `tagline`：标语。
- `vote_average`、`vote_count`：评分和投票数。

整理结果：

- `path`、`path_id`：最终整理后的路径。
- `video_file_name`、`video_file_id`：视频文件名和文件 ID。
- `video_pick_code`：115 / 百度网盘识别码。
- `video_open_list_sign`：OpenList 签名。
- `status`：媒体状态，存储值为 `scraped`、`scanned`、`renamed`。
- `subtitle_file_json`：字幕文件 JSON 列表。

运行时字段：

- `actors`、`director`、`origin_country`、`genres`、`subtitle_files`：运行时对象，不入库。

### `media_seasons`

剧集季表。

- `scrape_path_id`：来源刮削目录。
- `media_id`：所属媒体 ID。
- `season_number`：季号。
- `season_name`：季名称。
- `overview`：季简介。
- `number_of_episodes`：季内集数。
- `release_date`：发布日期。
- `path`、`path_id`：季目录路径和路径 ID。
- `poster_path`：季海报。
- `vote_average`：评分。
- `year`：年份。
- `status`：状态，和 `media.status` 共用同一套值。

### `media_episodes`

剧集集表。

- `scrape_path_id`：来源刮削目录。
- `media_id`：所属媒体 ID。
- `media_season_id`：所属季 ID。
- `episode_name`：集名称。
- `overview`：集简介。
- `poster_path`：集海报。
- `season_number`、`episode_number`：季号和集号。
- `release_date`：发布日期。
- `vote_average`、`vote_count`：评分和投票数。
- `year`：年份。
- `video_file_name`、`video_file_id`、`video_pick_code`、`video_open_list_sign`：视频定位信息。
- `status`：状态。
- `subtitle_file_json`：字幕文件 JSON。

运行时字段：

- `actors`：演员列表，不入库。

### `emby_config`

Emby 总配置表。

- `emby_url`：Emby 地址。
- `emby_api_key`：Emby API Key。
- `enable_delete_netdisk`：是否联动删除网盘文件。
- `enable_refresh_library`：是否刷新媒体库。
- `enable_media_notification`：是否发送媒体通知。
- `enable_extract_media_info`：是否提取媒体信息。
- `enable_auth`：是否启用 Emby Webhook 认证，默认启用。
- `sync_enabled`：是否启用同步。
- `sync_cron`：同步 Cron 表达式。
- `last_sync_time`：上次同步时间戳。
- `selected_libraries`：选中的媒体库 ID 列表，JSON 字符串。
- `sync_all_libraries`：是否同步所有媒体库。
- `enable_playback_overview`：播放通知是否显示剧情简介。
- `enable_playback_progress`：播放通知是否显示播放进度。

说明：

- `selected_libraries` 是 JSON 数组字符串，不是逗号分隔文本。
- `enable_playback_overview`、`enable_playback_progress` 是后加字段。

### `emby_media_items`

同步下来的 Emby 媒体项。

- `item_id`：Emby 媒体项 ID，唯一。
- `item_id_int`：数值化 ID，便于索引和排序。
- `server_id`：Emby Server ID。
- `name`：名称。
- `type`：类型，电影 / 集 / 文件夹等。
- `parent_id`：父节点 ID。
- `series_id`、`series_name`：剧集 ID 和名称。
- `season_id`、`season_name`：季 ID 和名称。
- `library_id`：所属媒体库 ID。
- `path`：媒体路径。
- `pick_code`：关联的 PickCode。
- `media_source_path`：媒体源路径。
- `index_number`、`parent_index_number`：集号 / 季号。
- `production_year`：年份。
- `premiere_date`：首播日期。
- `date_created`、`date_modified`：时间字符串。
- `date_created_time`、`date_modified_time`：对应时间戳。
- `is_folder`：是否文件夹。

### `emby_media_sync_files`

Emby 媒体项与同步文件的关联表。

- `sync_path_id`：同步目录 ID。
- `emby_item_id`：Emby 媒体项 ID。
- `sync_file_id`：同步文件 ID。
- `pick_code`：PickCode。

### `emby_libraries`

Emby 媒体库基础表。

- `name`：媒体库名称。
- `library_id`：Emby 媒体库 ID。
- `sync_path_id`：关联的同步目录 ID，`0` 表示未关联。

### `emby_library_sync_paths`

Emby 媒体库与同步目录的关联表。

- `library_id`：媒体库 ID。
- `sync_path_id`：同步目录 ID。
- `library_name`：媒体库名称。

### `emby_library_refresh_tasks`

Emby 媒体库刷新任务表。

- `library_id`：媒体库 ID，唯一。
- `library_name`：媒体库名称。
- `sync_path_ids_str`：关联的同步目录 ID 列表，JSON 字符串。
- `status`：任务状态，`pending`、`refreshing`、`completed`、`failed`、`cancelled`。
- `last_event_at`：最近一次事件时间戳。
- `refresh_after_at`：去抖后的刷新执行时间。
- `deadline_at`：最长等待时间。
- `last_checked_at`：最后检查时间。
- `last_refresh_at`：最后刷新时间。
- `error`：错误信息。

### `request_stats`

请求统计表。

- `request_time`：请求时间戳。
- `url`：请求 URL。
- `method`：请求方法。
- `duration`：响应时长，单位毫秒。
- `is_throttled`：是否被限流。
- `account_id`：关联账号 ID，默认 `0`。

### `db_download_tasks`

数据库下载队列表。

- `account_id`：账号 ID。
- `sync_file_id`：对应同步文件 ID。
- `source_type`：任务来源账号类型。
- `remote_file_id`：远程文件 ID 或下载链接。
- `file_name`：文件名。
- `remote_path`：远程路径。
- `local_full_path`：本地落盘路径。
- `source`：下载来源，见上面的 `download.source`。
- `status`：下载状态。
- `size`：文件大小。
- `start_time`、`end_time`：开始和结束时间戳。
- `error`：错误信息。
- `mtime`：下载完成后要回写的文件修改时间。
- `retry_count`：重试次数。
- `last_retry_time`：最近重试时间。

说明：

- `source` 的存储值在版本 `43` 迁移中从展示文案统一成稳定枚举值。
- `source_type` 仍然表示账号来源，不等于 `source`。

### `db_upload_tasks`

数据库上传队列表。

- `source`：上传来源，见上面的 `upload.source`。
- `account_id`：账号 ID。
- `sync_file_id`：同步文件 ID。
- `scrape_media_file_id`：刮削文件 ID。
- `source_type`：任务来源账号类型。
- `local_full_path`：本地完整路径。
- `remote_file_id`：远程文件 ID 或路径。
- `remote_path_id`：父目录 CID 或父路径。
- `file_name`：上传文件名。
- `status`：上传状态。
- `file_size`：文件大小。
- `error`：错误信息。
- `start_time`、`end_time`：开始和结束时间戳。
- `retry_count`：重试次数。
- `last_retry_time`：最近重试时间。
- `is_season_or_tvshow_file`：是否为剧集或电视剧文件。

### `backup_config`

自动备份配置表。

- `backup_enabled`：是否启用自动备份。
- `backup_cron`：备份 Cron 表达式。
- `backup_path`：备份存储路径。
- `backup_retention`：保留天数。
- `backup_max_count`：最多保留数量。
- `backup_compress`：是否压缩备份。

### `backup_record`

备份历史记录表。

- `task_id`：关联的任务 ID。
- `status`：状态，见上面的 `backup.status`。
- `file_path`：备份文件路径。
- `file_size`：备份文件大小，单位字节。
- `database_size`：数据库大小，单位字节。
- `table_count`：备份表数量。
- `backup_duration`：备份耗时，单位秒。
- `backup_type`：备份类型，`manual` 或 `auto`。
- `created_reason`：创建原因。
- `failure_reason`：失败原因。
- `compression_ratio`：压缩比。
- `is_compressed`：是否已压缩。
- `completed_at`：完成时间戳。

### `notification_channels`

通知渠道基础表。

- `channel_type`：渠道类型，当前内置实现包括 `telegram`、`meow`、`bark`、`serverchan`、`webhook`；允许创建多个相同类型渠道。
- `channel_name`：渠道名称。
- `description`：渠道说明。
- `is_enabled`：是否启用。
- `created_at` / `updated_at`：创建和更新时间，使用 `time.Time`。

### `telegram_channel_configs`

Telegram 渠道配置表。

- `channel_id`：关联的通知渠道 ID。
- `bot_token`：Bot Token。
- `chat_id`：Chat ID。
- `proxy_url`：代理地址。

### `meo_w_channel_configs`

MeoW 渠道配置表。

- `channel_id`：关联的通知渠道 ID。
- `nickname`：昵称。
- `endpoint`：请求地址，默认 `http://api.chuckfang.com`。

### `bark_channel_configs`

Bark 渠道配置表。

- `channel_id`：关联的通知渠道 ID。
- `device_key`：设备 Key。
- `server_url`：服务地址，默认 `https://api.day.app`。
- `sound`：提示音，默认 `alert`。
- `icon`：图标。

### `server_chan_channel_configs`

Server酱 渠道配置表。

- `channel_id`：关联的通知渠道 ID。
- `sc_key`：Server酱 Key。
- `endpoint`：服务地址，默认 `https://sc.ftqq.com`。

### `custom_webhook_channel_configs`

自定义 Webhook 渠道配置表。

- `channel_id`：关联的通知渠道 ID。
- `endpoint`：请求地址。
- `method`：请求方法，`GET` 或 `POST`。
- `template`：模板字符串，支持 `{{title}}`、`{{content}}`、`{{timestamp}}`、`{{image}}`。
- `format`：POST 格式，`json`、`form` 或 `text`。
- `auth_type`：鉴权方式，`none`、`bearer`、`basic`、`header`、`query`。
- `auth_token`：鉴权令牌。
- `auth_user` / `auth_pass`：Basic Auth 用户名和密码。
- `auth_header_key`：Header 模式下的头名。
- `auth_query_key`：Query 模式下的参数名。
- `headers`：额外请求头，JSON 对象字符串。
- `query_param`：GET 模式下承载模板的参数名，默认 `q`。

### `notification_rules`

通知规则表。

- `channel_id`：关联的通知渠道 ID。
- `event_type`：事件类型，见上面的 `notification.type`。
- `is_enabled`：是否启用。

## 备份与恢复

备份和恢复逻辑位于 `backend/internal/backup/`，同样依赖 `AllTables`：

- 备份时按表导出为 JSON Lines，再打包成压缩包。
- 恢复时会解压备份包，逐表删除旧表并重建，再导入数据。
- 这类逻辑依赖当前模型定义，所以新增或修改表字段后，备份和恢复行为也会随之变化。
- 前端的两处恢复入口都会在确认弹窗提示：恢复成功后请重启服务，

这是一套全量表级方案，不是增量备份，也不是时间点恢复。
