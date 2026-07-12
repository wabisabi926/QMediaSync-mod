# 任务来源枚举

任务队列的 `source`、部分 `source_type` 字段会写入数据库，并参与查询、去重和分支判断。同步队列的 `task_type` 不写入数据库，但参与队列路由、去重 key、取消和状态查询。

后端业务标识使用稳定机器值，展示层负责把机器值映射为用户可读文案。

下载任务状态事件中的 `DownloadTaskStatusChangedPayload.Source` 也使用同一套机器值。事件消费者如果需要展示来源，必须先映射为展示文案。

`DownloadTaskStatusChangedPayload.sync_path_id` 表示 STRM 下载任务所属同步目录 ID。Emby 刷新协调器优先使用该字段判断媒体库关联目录的下载队列是否完成；字段为 `0` 或 `NULL` 时兼容使用 `sync_file_id` 查询。同一 Emby 媒体库关联的任一同步目录仍在等待、运行或存在未完成下载任务时，该媒体库刷新任务会继续等待。item 级刷新任务使用 `task_key=item:<item_id>` 去重，`library_id` 仅保存真实媒体库 ID 或为空；下载事件唤醒和清空 pending 下载任务取消时会按真实媒体库 ID 匹配，优先使用 `fallback_library_id`，为空时按任务 `item_ids` 查询本地 Emby item 所属媒体库。

## 下载任务

| 字段 | 存储值 | 展示文案 | 用途 |
| --- | --- | --- | --- |
| `db_download_tasks.source` | `strm_sync` | `STRM 同步` | STRM 同步产生的下载任务 |
| `db_download_tasks.source` | `local_file` | `本地文件` | 本地文件复制任务 |
| `db_download_tasks.source` | `emby_media` | `Emby 媒体信息提取` | Emby 媒体信息提取触发任务 |
| `db_download_tasks.source_type` | `emby_media` | `Emby 媒体信息提取` | Emby 媒体信息提取专用来源类型 |

## 上传任务

| 字段 | 存储值 | 展示文案 | 用途 |
| --- | --- | --- | --- |
| `db_upload_tasks.source` | `strm_sync` | `STRM 同步` | STRM 同步产生的上传任务 |
| `db_upload_tasks.source` | `scrape_organize` | `刮削整理` | 刮削整理产生的上传任务 |
| `db_upload_tasks.source` | `directory_monitor` | `目录监控上传` | 目录监控发现本地文件后产生的上传任务 |

## 同步队列任务类型

| 字段 | 存储值 | 展示文案 | 用途 |
| --- | --- | --- | --- |
| `SyncTaskTypeStrm` | `strm_sync` | `STRM 同步` | STRM 同步队列任务 |
| `SyncTaskTypeScrape` | `scrape_organize` | `刮削整理` | 刮削整理队列任务 |

## 维护约束

- 不要把展示文案直接写入任务来源字段或同步队列任务类型。
- 新增任务来源或队列任务类型时，同步更新后端常量、前端展示映射和本文档。
- 会写入数据库的历史数据通过 `backend/internal/models/migrator.go` 迁移到当前存储值。
- 仅存在于内存队列的任务类型不需要数据库迁移，但需要测试队列 key 和状态输出是否使用机器值。
- `frontend/src/utils/sourceTypeUtils.ts` 面向同步目录和账号来源；任务队列来源使用 `frontend/src/utils/taskSourceUtils.ts`，不要混用两套 `local` 文案。
- `MQ的媒体库` 是 115 授权应用名，不属于任务来源枚举。
