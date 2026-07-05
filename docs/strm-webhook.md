# STRM Webhook

本文说明外部程序通过 Webhook 请求 QMediaSync 创建 STRM 生成任务的接口边界。Webhook 请求只负责入队，实际 STRM 写入、`SyncFile` 更新、同名元数据下载和 Emby 刷新提交都由后台 STRM worker 处理。

## 入口和鉴权

接口地址：

```http
POST /api/strm/webhook
Content-Type: application/json
X-API-Key: qms_xxxxxxxxxxxxxxxxxxxxxxxx
```

接口不使用浏览器登录态，也不需要 CSRF。鉴权只接受 API Key：

- 推荐使用 `X-API-Key` 请求头。
- 仅在调用方无法设置请求头时，使用 `?api_key=` 查询参数。
- API Key 无效或缺失时返回 HTTP `401`。

API Key 在 Web 页面「系统设置 - API Key」中创建。完整密钥只会在创建响应中返回一次，后端只保存哈希。

## 处理边界

- 请求必须提供 `sync_path_id`，并且同步目录必须存在。
- 当前 Webhook 按 115 远端文件详情解析文件信息，`path` 和 `directory_path` 都表示 115 远端路径。
- 禁止通过 `local_path` 指定本地写入位置；顶层请求和批量项中的 `local_path` 都会被拒绝。
- 本地 STRM 输出路径只能由 `sync_path_id` 对应同步目录的配置计算。
- 文件级请求入队前会按 `file_id` 或 `path + file_name` 查询远端详情，并以解析后的真实远端路径再次校验同步目录边界。
- 目录级请求只创建目录扫描父任务，HTTP 请求内不同步展开完整目录。
- 合法请求只创建 `strm_generation_tasks`，不会在 HTTP 请求内直接写 STRM，也不会直接请求 Emby。
- `download_meta` 和 `refresh_emby` 默认关闭；调用方必须显式开启需要的后处理。

## 请求动作

`action` 支持三类值：

| action | 说明 |
| --- | --- |
| `file` | 创建单文件 STRM 生成任务。 |
| `batch_files` | 创建批量父任务，再为每个合法 `items[]` 项创建单文件子任务。单项失败不会影响其他合法项入队。 |
| `directory_scan` | 创建目录扫描父任务，后续由 STRM worker 异步枚举远端目录并为视频文件创建子任务。 |

`action` 为空时，后端按字段自动判断：

- 有 `items` 时视为 `batch_files`。
- 有 `directory_id` 或 `directory_path` 时视为 `directory_scan`。
- 其他情况视为 `file`。

## 请求字段

通用字段：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `sync_path_id` | 是 | 同步目录 ID。 |
| `action` | 否 | `file`、`batch_files` 或 `directory_scan`。 |
| `download_meta` | 否 | 是否下载本次视频强相关的同名元数据，默认 `false`。 |
| `refresh_emby` | 否 | 是否在 STRM 变更或新增元数据下载任务后提交 Emby 刷新目标，默认 `false`。 |

`download_meta` 和 `refresh_emby` 是请求级字段。`batch_files` 的 `items[]` 内禁止放这两个开关；如果任一 item 包含 `download_meta` 或 `refresh_emby`，整个请求返回 HTTP `400`。批量和目录扫描会按整个请求统一控制，所有子任务继承父任务开关。

单文件字段：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `path` | 条件必填 | 文件所在 115 远端目录。使用 `path + file_name` 定位时必填。 |
| `file_name` | 条件必填 | 文件名。使用 `path + file_name` 定位时必填。 |
| `file_id` | 条件必填 | 115 文件 ID。与 `path + file_name` 至少提供一组；提供后后端优先用它查询真实远端文件信息。 |
| `pick_code` | 否 | 只能作为辅助字段，不能单独定位文件。 |
| `parent_id` | 否 | 父目录 ID；远端详情解析成功后会以解析结果为准。 |
| `file_size` | 否 | 文件大小，单位字节；远端详情解析成功后会以解析结果为准。 |
| `sha1` | 否 | 文件 SHA1；远端详情解析成功后会以解析结果为准。 |
| `mtime` | 否 | 文件远端更新时间戳；远端详情解析成功后会以解析结果为准。 |

批量请求使用顶层 `items` 数组，数组项字段与单文件字段一致。`items` 不能为空。

目录扫描字段：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `directory_id` | 条件必填 | 115 目录 ID。与 `directory_path` 至少提供一个。 |
| `directory_path` | 条件必填 | 115 远端目录路径。提供时必须位于同步目录的 `remote_path` 下。 |

## 元数据下载

`download_meta=true` 时只下载和本次生成视频强相关的元数据。对视频文件名去掉扩展名得到 `base`，只匹配：

- `base + 元数据扩展名`
- `base + "-thumb" + 元数据扩展名`

例如视频文件是：

```text
古诺希亚 - S01E18 - ANi.mp4
```

只会匹配类似：

```text
古诺希亚 - S01E18 - ANi.nfo
古诺希亚 - S01E18 - ANi.srt
古诺希亚 - S01E18 - ANi-thumb.jpg
古诺希亚 - S01E18 - ANi-thumb.png
```

不会下载同目录下其他海报、fanart、season 图、全局 NFO 等文件。Webhook 元数据下载也不会删除本地多余元数据，不会上传本地元数据。

元数据扩展名沿用当前同步目录的 STRM 配置：同步目录自定义配置优先，未自定义时回退全局配置。只有本地缺失的匹配元数据会创建下载任务，下载仍走现有下载队列。后台 STRM worker 也会校验任务来源，只有 Webhook file 任务会响应该开关，上传完成、远端已存在等非 Webhook STRM 任务即使意外写入该字段也不会触发这套同名元数据下载规则。

## Emby 刷新

`refresh_emby=true` 时，文件任务只有在 STRM 发生变更或新增元数据下载任务后才解析并提交刷新目标。

- `file`：单文件任务完成后提交目标，但仍进入现有 Emby 刷新协调器防抖。
- `batch_files`：所有子任务完成或失败后，由批量父任务统一提交一次已收集目标集合。
- `directory_scan`：目录展开出的所有子任务完成或失败后，由目录扫描父任务统一提交一次已收集目标集合。

刷新目标优先保持 item 精度。每个发生变化的文件会先解析为 Movie、Episode、Season、Series 等 item；解析不到可靠 item 时回退同步目录关联媒体库。目标集合会去重，同一媒体库内如果出现 library fallback，该媒体库内其他 item 目标会被覆盖，不再单独提交；不同媒体库互不影响。

提交后只写入 `emby_library_refresh_tasks`，不在 STRM worker 内直接请求 Emby。刷新协调器会使用现有 `RefreshAfterAt = LastEventAt + 10s` 防抖窗口；如果同一 `sync_path_id` 仍有 `pending` 或 `downloading` 下载任务，协调器会继续等待下载完成后再刷新。

上传完成、远端已存在等非 Webhook STRM 任务保持原有行为：STRM 变更后可逐文件提交 Emby 刷新，不受 Webhook 默认开关影响。

## 响应

成功接收请求时返回 HTTP `200`，业务响应的 `data` 包含：

| 字段 | 说明 |
| --- | --- |
| `request_id` | 本次 Webhook 请求 ID，格式为 `strm_` 加随机字符串。 |
| `task_ids` | 本次接受的 STRM 生成任务 ID 列表。`batch_files` 返回合法文件子任务 ID，不返回批量父任务 ID。 |
| `accepted_count` | 成功入队或命中已有活动任务的数量。 |
| `failed_count` | 批量请求中失败项数量。 |
| `results` | 逐项结果，`index` 从 0 开始，包含 `accepted`、`task_id` 或 `error`。 |

单文件和目录扫描请求如果校验失败，会返回 HTTP `400`。批量请求中单项失败不会导致整个请求失败，只会在对应 `results` 项中写入错误；但 `items[]` 内放入请求级开关属于整批请求错误，会直接返回 HTTP `400`。

示例响应：

```json
{
  "code": 200,
  "message": "STRM 生成任务已接收",
  "data": {
    "request_id": "strm_abcdefghijklmnop",
    "task_ids": [123],
    "accepted_count": 1,
    "failed_count": 0,
    "results": [
      {
        "index": 0,
        "accepted": true,
        "task_id": 123
      }
    ]
  }
}
```

## 幂等和重试

Webhook 入队会为请求生成 `request_hash`，哈希包含请求动作、远端定位信息和请求级开关：

- `pending` 或 `running` 状态的相同请求会复用已有任务，不重复创建。
- 历史任务如果已经 `failed`、`completed` 或 `cancelled`，再次提交相同请求会归档旧请求哈希并创建新的待处理任务。
- worker 只自动领取 `pending` 任务；执行失败会把任务标记为 `failed`，递增 `retry_count` 并写入 `last_error`。
- `batch_files` 父任务状态为 `completed` 只表示父记录本身不需要 worker 执行，不表示整个批次已全部处理完成；批次完成度以 `total_items`、`accepted_items`、`failed_items` 和 `refresh_submitted` 判断。
- `directory_scan` 父任务展开完成后记录 `total_items`；子任务后续完成或失败时累计 `accepted_items`、`failed_items`、`changed_items` 和 `new_meta_items`。

## 示例

单文件请求主示例使用 `path + file_name`：

```json
{
  "sync_path_id": 1,
  "action": "file",
  "download_meta": true,
  "refresh_emby": true,
  "path": "/剧集/示例剧/S01",
  "file_name": "示例剧 - S01E01.mkv"
}
```

也可以可选提供 `file_id`，后端会优先用它查询真实远端文件信息：

```json
{
  "sync_path_id": 1,
  "action": "file",
  "download_meta": true,
  "refresh_emby": true,
  "file_id": "115-file-id",
  "path": "/剧集/示例剧/S01",
  "file_name": "示例剧 - S01E01.mkv"
}
```

批量文件请求：

```json
{
  "sync_path_id": 1,
  "action": "batch_files",
  "download_meta": false,
  "refresh_emby": false,
  "items": [
    {
      "path": "/剧集/示例剧/S01",
      "file_name": "示例剧 - S01E01.mkv",
      "file_id": "115-file-id-1"
    },
    {
      "path": "/剧集/示例剧/S01",
      "file_name": "示例剧 - S01E02.mkv",
      "file_id": "115-file-id-2"
    }
  ]
}
```

目录扫描请求：

```json
{
  "sync_path_id": 1,
  "action": "directory_scan",
  "download_meta": true,
  "refresh_emby": true,
  "directory_path": "/剧集/示例剧/S01"
}
```

`curl` 示例：

```bash
curl -X POST 'http://127.0.0.1:12333/api/strm/webhook' \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: qms_xxxxxxxxxxxxxxxxxxxxxxxx' \
  -d '{
    "sync_path_id": 1,
    "action": "file",
    "download_meta": true,
    "refresh_emby": true,
    "path": "/剧集/示例剧/S01",
    "file_name": "示例剧 - S01E01.mkv"
  }'
```
