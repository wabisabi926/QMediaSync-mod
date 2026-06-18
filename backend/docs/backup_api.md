# 备份恢复模块 API 接口文档

## 基础信息

- **基础路径**: `/api`
- **认证方式**: JWT Token 或 API Key
- **请求头**: 
  - `Authorization: Bearer <token>` 或
  - `?api_key=<api_key>`

## 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 响应码说明

| Code | 说明 |
|------|------|
| 0 | 成功 |
| 1 | 请求错误 |
| 2 | 未授权 |

---

## 接口列表

### 1. 创建手动备份

**POST** `/backup/create`

创建一次手动备份数据库。

**请求体**:
```json
{
  "reason": "手动备份"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reason | string | 否 | 备份原因，默认为"手动备份" |

**响应示例**:
```json
{
  "code": 0,
  "message": "备份创建成功",
  "data": {
    "id": 1,
    "created_at": 1700000000,
    "updated_at": 1700000000,
    "status": "completed",
    "file_path": "/path/to/backups/backup_manual_20231115_120000.sql.zip",
    "file_size": 1024000,
    "database_size": 5000000,
    "table_count": 15,
    "backup_duration": 5,
    "backup_type": "manual",
    "created_reason": "手动备份",
    "compression_ratio": 0.2,
    "is_compressed": 1,
    "completed_at": 1700000005
  }
}
```

**错误响应**:
```json
{
  "code": 1,
  "message": "备份任务正在运行中",
  "data": null
}
```

---

### 2. 获取备份列表

**GET** `/backup/list`

获取备份记录列表，支持分页和类型筛选。

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量（最大100） |
| type | string | 否 | all | 备份类型：all/manual/auto |

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "created_at": 1700000000,
        "status": "completed",
        "file_path": "/path/to/backup.sql.zip",
        "file_size": 1024000,
        "backup_type": "manual",
        "backup_duration": 5,
        "created_reason": "手动备份"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 3. 获取备份记录详情

**GET** `/backup/records/:id`

获取指定备份记录的详细信息。

**路径参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 备份记录ID |

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "created_at": 1700000000,
    "updated_at": 1700000000,
    "task_id": 0,
    "status": "completed",
    "file_path": "/path/to/backup.sql.zip",
    "file_size": 1024000,
    "database_size": 5000000,
    "table_count": 15,
    "backup_duration": 5,
    "backup_type": "manual",
    "created_reason": "手动备份",
    "failure_reason": "",
    "compression_ratio": 0.2,
    "is_compressed": 1,
    "completed_at": 1700000005
  }
}
```

---

### 4. 删除备份记录

**DELETE** `/backup/records/:id`

删除指定的备份记录及其对应的物理备份文件。

**路径参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 备份记录ID |

**响应示例**:
```json
{
  "code": 0,
  "message": "备份已删除",
  "data": null
}
```

**错误响应**:
```json
{
  "code": 1,
  "message": "备份任务正在运行中，无法删除",
  "data": null
}
```

---

### 5. 从备份记录恢复

**POST** `/backup/restore`

根据备份记录ID恢复数据库。

**请求体**:
```json
{
  "record_id": 1
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| record_id | uint | 是 | 要恢复的备份记录ID |

**响应示例**:
```json
{
  "code": 0,
  "message": "数据恢复成功",
  "data": null
}
```

**注意事项**: 
- 恢复前会自动创建当前数据库的备份
- 恢复过程会暂停所有任务队列和定时任务
- 恢复完成后自动恢复队列和定时任务

---

### 6. 上传文件并恢复

**POST** `/backup/upload-restore`

上传SQL备份文件并执行恢复。

**请求格式**: `multipart/form-data`

**表单参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 备份文件（.sql 或 .zip 格式） |

**响应示例**:
```json
{
  "code": 0,
  "message": "数据恢复成功",
  "data": null
}
```

**错误响应**:
```json
{
  "code": 1,
  "message": "仅支持.sql和.zip格式的备份文件",
  "data": null
}
```

---

### 7. 下载备份文件

**GET** `/backup/download/:id`

下载指定的备份文件。

**路径参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 备份记录ID |

**响应**: 
- Content-Type: `application/octet-stream`
- Content-Disposition: `attachment; filename=<filename>`

---

### 8. 获取备份配置

**GET** `/backup/config`

获取当前备份配置信息。

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "created_at": 1700000000,
    "updated_at": 1700000000,
    "backup_enabled": 1,
    "backup_cron": "0 3 * * *",
    "backup_path": "",
    "backup_retention": 7,
    "backup_max_count": 10,
    "backup_compress": 1
  }
}
```

---

### 9. 更新备份配置

**PUT** `/backup/config`

更新备份配置，支持动态更新定时任务。

**请求体**:
```json
{
  "backup_enabled": 1,
  "backup_cron": "0 3 * * *",
  "backup_retention": 7,
  "backup_max_count": 10,
  "backup_compress": 1
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| backup_enabled | int | 是 | 是否启用自动备份，0=禁用，1=启用 |
| backup_cron | string | 是 | Cron表达式，如 `0 3 * * *` 表示每天凌晨3点 |
| backup_retention | int | 是 | 备份保留天数 |
| backup_max_count | int | 是 | 最多保留备份数量 |
| backup_compress | int | 是 | 是否压缩，0=不压缩，1=压缩 |

**响应示例**:
```json
{
  "code": 0,
  "message": "配置已更新",
  "data": null
}
```

---

### 10. 获取备份状态

**GET** `/backup/status`

获取当前备份服务状态。

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "is_running": false,
    "backup_dir": "/path/to/config/backups",
    "config": {
      "id": 1,
      "backup_enabled": 1,
      "backup_cron": "0 3 * * *",
      "backup_retention": 7,
      "backup_max_count": 10,
      "backup_compress": 1
    }
  }
}
```

---

### 11. 取消正在运行的备份

**POST** `/backup/cancel`

取消当前正在运行的备份任务。

**响应示例**:
```json
{
  "code": 0,
  "message": "已发送取消信号",
  "data": null
}
```

**无任务时响应**:
```json
{
  "code": 0,
  "message": "没有正在运行的备份任务",
  "data": null
}
```

---

## 数据模型

### BackupRecord 备份记录

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 记录ID |
| created_at | int64 | 创建时间戳 |
| updated_at | int64 | 更新时间戳 |
| task_id | uint | 关联任务ID |
| status | string | 状态：pending/running/completed/failed/cancelled/timeout |
| file_path | string | 备份文件路径 |
| file_size | int64 | 文件大小（字节） |
| database_size | int64 | 原始数据库大小（字节） |
| table_count | int | 表数量 |
| backup_duration | int64 | 备份耗时（秒） |
| backup_type | string | 类型：manual/auto |
| created_reason | string | 创建原因 |
| failure_reason | string | 失败原因 |
| compression_ratio | float64 | 压缩比 |
| is_compressed | int | 是否压缩：0/1 |
| completed_at | int64 | 完成时间戳 |

### BackupConfig 备份配置

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 配置ID |
| created_at | int64 | 创建时间戳 |
| updated_at | int64 | 更新时间戳 |
| backup_enabled | int | 是否启用自动备份：0/1 |
| backup_cron | string | 定时任务Cron表达式 |
| backup_path | string | 备份存储路径 |
| backup_retention | int | 备份保留天数 |
| backup_max_count | int | 最大备份数量 |
| backup_compress | int | 是否压缩：0/1 |

---

## Cron 表达式说明

格式：`分 时 日 月 周`

常用示例：

| 表达式 | 说明 |
|--------|------|
| `0 3 * * *` | 每天凌晨3点 |
| `0 */6 * * *` | 每6小时 |
| `0 0 * * 0` | 每周日午夜 |
| `0 2 1 * *` | 每月1日凌晨2点 |
| `30 4 * * 1-5` | 周一到周五凌晨4:30 |

---

## 备份文件格式

### 文件命名规则

- 手动备份: `backup_manual_YYYYMMDD_HHMMSS.sql.zip`
- 自动备份: `backup_auto_YYYYMMDD_HHMMSS.sql.zip`

### 压缩格式

当 `backup_compress=1` 时，备份文件为ZIP格式，内含 `backup.sql` 文件。

### SQL文件结构

```sql
-- QMediaSync Database Backup
-- Generated at: 2024-01-15 12:00:00
-- Database Engine: sqlite

PRAGMA foreign_keys=OFF;

DROP TABLE IF EXISTS table_name;
CREATE TABLE table_name (...);
INSERT INTO table_name (col1, col2) VALUES (val1, val2);
```

---

## 错误处理

### 常见错误码

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| 备份任务正在运行中 | 上一次备份尚未完成 | 等待完成或调用取消接口 |
| 备份文件不存在 | 物理文件被手动删除 | 删除该记录重新备份 |
| 仅支持.sql和.zip格式 | 上传了不支持的文件格式 | 使用正确格式上传 |
| 备份记录不存在 | 指定的ID无对应记录 | 检查ID是否正确 |

---

## 最佳实践

1. **定期备份**: 建议配置每日自动备份，选择业务低峰期（如凌晨3点）

2. **保留策略**: 
   - 保留天数建议7-14天
   - 最大备份数建议10-20个

3. **恢复操作**:
   - 恢复前系统会自动创建当前数据备份
   - 恢复期间系统会暂停所有同步任务
   - 建议在业务低峰期执行恢复

4. **存储空间**: 
   - 启用压缩可节省约80%存储空间
   - 定期检查备份目录空间
