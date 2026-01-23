# Emby 同步和配置 API 文档

## 功能概述

Emby 功能模块提供完整的 Emby 服务器集成、媒体同步、库管理和选择码提取等功能，实现 Emby 和 115 网盘的无缝协作。

### 核心特性

- ✅ **配置管理**: 支持 Emby 服务器地址、API Key 等配置
- ✅ **媒体同步**: 自动或手动同步 Emby 媒体库中的所有项目
- ✅ **选择码提取**: 通过 PlaybackInfo 接口自动提取 115 网盘选择码
- ✅ **库关联**: 支持多个 Emby 媒体库关联到不同的同步目录
- ✅ **媒体查询**: 分页查询同步后的媒体项目，支持按库和类型筛选
- ✅ **定时同步**: 支持 cron 表达式配置定时自动同步
- ✅ **并发控制**: 同时只允许一个同步任务运行，防止冲突
- ✅ **实时反馈**: 获取同步状态和历史信息

### 约束和限制

1. **同时只能有一个 Emby 同步任务运行**
2. **同步任务最长执行时间：30 分钟（推荐）**
3. **PlaybackInfo 调用超时：30 秒**
4. **并发工作线程：10 个**
5. **媒体项目分页大小：50-200 条**
6. **cron 表达式验证：必须有效且能解析**

---

## API 接口列表

### 基础信息

- **Base URL**: `/api`
- **认证方式**: JWT Token（通过 `Authorization: Bearer <token>` 请求头）
- **响应格式**: JSON
- **内容类型**: `application/json`

#### 通用响应格式

```json
{
  "code": 200,           // 200: 成功, 400: 失败
  "message": "操作成功",  // 消息提示
  "data": {}             // 响应数据，可能为 null
}
```

---

## 1. Emby 配置管理

### 1.1 获取 Emby 配置

**接口**: `GET /api/setting/emby-config`

**描述**: 获取当前 Emby 配置信息

**请求参数**: 无

**响应示例 (配置存在)**:

```json
{
  "code": 200,
  "message": "获取Emby配置成功",
  "data": {
    "exists": true,
    "config": {
      "id": 1,
      "emby_url": "http://192.168.1.100:8096",
      "emby_api_key": "your-api-key-here",
      "enable_delete_netdisk": 0,
      "enable_refresh_library": 1,
      "enable_media_notification": 1,
      "enable_extract_media_info": 1,
      "sync_enabled": 1,
      "sync_cron": "0 2 * * *",
      "last_sync_time": 1706000000,
      "created_at": 1705999000,
      "updated_at": 1705999000
    }
  }
}
```

**响应示例 (配置不存在)**:

```json
{
  "code": 200,
  "message": "获取Emby配置成功",
  "data": {
    "exists": false
  }
}
```

---

### 1.2 更新 Emby 配置

**接口**: `POST /api/setting/emby-config`

**描述**: 创建或更新 Emby 配置

**请求参数**:

```json
{
  "emby_url": "http://192.168.1.100:8096",        // 必填: Emby 服务器地址
  "emby_api_key": "your-api-key-here",            // 必填: Emby API Key
  "enable_delete_netdisk": 0,                     // 可选: 是否在删除 Emby 项目时删除网盘文件 (0/1)
  "enable_refresh_library": 1,                    // 可选: 是否在同步时刷新 Emby 媒体库 (0/1)
  "enable_media_notification": 1,                 // 可选: 是否发送媒体同步通知 (0/1)
  "enable_extract_media_info": 1,                 // 可选: 是否提取媒体额外信息 (0/1)
  "sync_enabled": 1,                              // 必填: 是否启用自动同步 (0/1)
  "sync_cron": "0 2 * * *"                        // 可选: 同步 cron 表达式（当 sync_enabled=1 时必填）
}
```

**Cron 表达式说明**:

```
格式: * * * * *
     ┬ ┬ ┬ ┬ ┬
     │ │ │ │ │
     │ │ │ │ └─ 星期几 (0-6, 0=星期日)
     │ │ │ └─── 月份 (1-12)
     │ │ └───── 日期 (1-31)
     │ └─────── 小时 (0-23)
     └───────── 分钟 (0-59)

常见示例:
"0 2 * * *"     - 每天凌晨2点
"0 */4 * * *"   - 每4小时执行一次
"0 0 * * 0"     - 每周日凌晨0点
"0 3 1 * *"     - 每月1号凌晨3点
"*/30 * * * *"  - 每30分钟执行一次
```

**响应示例 (成功)**:

```json
{
  "code": 200,
  "message": "Emby配置更新成功",
  "data": null
}
```

**错误响应示例**:

```json
{
  "code": 400,
  "message": "cron表达式格式无效",
  "data": null
}
```

**验证规则**:

- Emby URL 不能为空
- API Key 不能为空
- 启用自动同步时 cron 表达式必须有效且可解析
- 表达式必须是合法的 cron 格式

**前端注意事项**:

1. 更新配置后会自动重启定时任务
2. 建议使用 cron 编辑器帮助用户输入（如 `vue-cron` 或 `react-cron`）
3. 测试 Emby 连接时调用获取同步状态接口
4. 显示最后同步时间，帮助用户了解同步状态

---

## 2. Emby 同步管理

### 2.1 手动启动同步

**接口**: `POST /api/emby/sync/start`

**描述**: 手动触发一次 Emby 媒体同步

**请求参数**: 无

**响应示例 (成功)**:

```json
{
  "code": 200,
  "message": "Emby同步任务已启动",
  "data": null
}
```

**错误响应示例**:

```json
{
  "code": 400,
  "message": "已有Emby同步任务正在运行，请稍候",
  "data": null
}
```

```json
{
  "code": 400,
  "message": "未找到Emby配置，请先配置",
  "data": null
}
```

```json
{
  "code": 400,
  "message": "Emby Url或ApiKey为空",
  "data": null
}
```

**说明**:

- 同步是异步操作，调用后返回立即返回
- 同时只能有一个同步任务运行，如果已有任务会返回错误
- 建议在用户界面禁用同步按钮直到任务完成
- 可以通过查询同步状态接口监控进度

---

### 2.2 获取同步状态

**接口**: `GET /api/emby/sync/status`

**描述**: 查询 Emby 同步的当前状态和统计信息

**请求参数**: 无

**响应示例 (已配置)**:

```json
{
  "code": 200,
  "message": "获取同步状态成功",
  "data": {
    "exists": true,
    "last_sync_time": 1706000000,    // Unix 时间戳，0 表示从未同步
    "total_items": 1256,              // 已同步的总媒体项目数
    "sync_enabled": 1                 // 是否启用自动同步 (0/1)
  }
}
```

**响应示例 (未配置)**:

```json
{
  "code": 200,
  "message": "尚未配置Emby",
  "data": {
    "exists": false
  }
}
```

**时间戳转换参考 (JavaScript)**:

```javascript
// Unix 时间戳转格式化日期
function formatTimestamp(timestamp) {
  if (timestamp === 0) return '从未同步';
  const date = new Date(timestamp * 1000);
  return date.toLocaleString('zh-CN');
}

// 获取最后同步距离现在的时间
function getRelativeTime(timestamp) {
  if (timestamp === 0) return '从未同步';
  const now = Math.floor(Date.now() / 1000);
  const seconds = now - timestamp;
  if (seconds < 60) return '刚刚';
  if (seconds < 3600) return Math.floor(seconds / 60) + '分钟前';
  if (seconds < 86400) return Math.floor(seconds / 3600) + '小时前';
  return Math.floor(seconds / 86400) + '天前';
}
```

---

## 3. Emby 媒体项目查询

### 3.1 分页查询媒体项目

**接口**: `GET /api/emby/media`

**描述**: 分页查询已同步的 Emby 媒体项目，支持按媒体库和类型筛选

**请求参数**:

| 参数 | 类型 | 必填 | 说明 | 默认值 |
|-----|------|-----|------|--------|
| page | int | 否 | 页码 | 1 |
| page_size | int | 否 | 每页数量 (50-200) | 50 |
| library_id | string | 否 | 媒体库 ID，留空则查询全部 | 全部 |
| type | string | 否 | 媒体类型 (Movie/Series/Season/Episode) | 全部 |

**请求示例**:

```
GET /api/emby/media?page=1&page_size=50&library_id=7&type=Movie
```

**响应示例**:

```json
{
  "code": 200,
  "message": "获取Emby媒体项成功",
  "data": {
    "total": 1256,
    "items": [
      {
        "id": 1,
        "item_id": "65102",
        "server_id": "",
        "name": "小城大事",
        "type": "Series",
        "parent_id": "",
        "series_id": "",
        "library_id": "7",
        "path": "/mnt/emby/series/小城大事",
        "pick_code": "a706hdog2st3zhfw6",
        "media_source_path": "http://qms.mqfamily.top:12333/115/url/...",
        "index_number": 0,
        "parent_index_number": 0,
        "production_year": 2026,
        "premiere_date": "2026-01-15",
        "date_created": "2026-01-15T10:30:00Z",
        "date_modified": "2026-01-15T10:30:00Z",
        "is_folder": true,
        "created_at": "2026-01-23T21:00:00Z",
        "updated_at": "2026-01-23T21:00:00Z"
      },
      {
        "id": 2,
        "item_id": "65103",
        "name": "小城大事 - S01E01",
        "type": "Episode",
        "parent_id": "65102",
        "series_id": "65102",
        "library_id": "7",
        "path": "/mnt/emby/series/小城大事/Season 1/...",
        "pick_code": "a707hdog2st3zhfw7",
        "media_source_path": "http://qms.mqfamily.top:12333/115/url/video.mkv?pickcode=a707hdog2st3zhfw7",
        "index_number": 1,
        "parent_index_number": 1,
        "production_year": 2026,
        "premiere_date": "2026-01-15",
        "date_created": "2026-01-15T10:35:00Z",
        "date_modified": "2026-01-15T10:35:00Z",
        "is_folder": false
      }
    ]
  }
}
```

**媒体类型说明**:

- **Movie**: 电影
- **Series**: 电视剧（整部剧集）
- **Season**: 季（如"第1季"）
- **Episode**: 集（如"第1集"）
- **BoxSet**: 系列合集
- **MusicAlbum**: 音乐专辑

**字段说明**:

| 字段 | 说明 |
|-----|------|
| item_id | Emby 项目的唯一 ID |
| name | 项目名称 |
| type | 媒体类型 |
| library_id | 所属媒体库 ID |
| pick_code | 115 网盘选择码（关键，用于直链获取） |
| path | Emby 中的路径 |
| media_source_path | PlaybackInfo 中的媒体源路径 |
| is_folder | 是否为文件夹 |
| parent_id | 上级项目 ID（对于 Episode，指向 Season） |
| series_id | 所属剧集 ID（对于 Episode，指向 Series） |

**前端注意事项**:

1. 分页大小限制在 50-200 之间
2. 默认分页大小为 50，大数据集建议使用虚拟滚动
3. 使用 pick_code 可以直接获取 115 网盘直链
4. parent_id 和 series_id 可用于构建树形结构

---

## 4. Emby 媒体库与同步目录关联

### 4.1 获取所有关联

**接口**: `GET /api/emby/library-sync-paths`

**描述**: 获取 Emby 媒体库和同步目录的关联关系

**请求参数**: 无

**响应示例**:

```json
{
  "code": 200,
  "message": "获取成功",
  "data": [
    {
      "id": 1,
      "library_id": "7",
      "library_name": "Movies",
      "sync_path_id": 1,
      "created_at": "2026-01-23T20:00:00Z",
      "updated_at": "2026-01-23T20:00:00Z"
    },
    {
      "id": 2,
      "library_id": "8",
      "library_name": "TV Series",
      "sync_path_id": 2,
      "created_at": "2026-01-23T20:05:00Z",
      "updated_at": "2026-01-23T20:05:00Z"
    }
  ]
}
```

**说明**:

- 一个媒体库可以关联到多个同步目录（用于多网盘存储）
- 一个同步目录也可以关联多个媒体库

---

### 4.2 创建或更新关联

**接口**: `POST /api/emby/library-sync-paths`

**描述**: 创建或更新 Emby 媒体库与同步目录的关联（去重，若已存在则跳过）

**请求参数**:

```json
{
  "library_id": "7",              // 必填: Emby 媒体库 ID（字符串）
  "sync_path_id": 1,              // 必填: 同步目录 ID（整数）
  "library_name": "Movies"        // 可选: 媒体库名称（用于显示）
}
```

**响应示例**:

```json
{
  "code": 200,
  "message": "更新成功",
  "data": null
}
```

**错误响应示例**:

```json
{
  "code": 400,
  "message": "更新关联失败: sync_path_id 不存在",
  "data": null
}
```

**前端注意事项**:

1. 需要先获取可用的同步目录列表（调用 `/api/sync/path-list` 或类似接口）
2. library_id 是字符串，sync_path_id 是整数
3. 同一个库和同步目录的组合只会被保存一次

---

### 4.3 删除关联

**接口**: `DELETE /api/emby/library-sync-paths`

**描述**: 删除指定 Emby 媒体库的同步关联

**请求参数** (Query String):

| 参数 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| library_id | string | 是 | Emby 媒体库 ID |

**请求示例**:

```
DELETE /api/emby/library-sync-paths?library_id=7
```

**响应示例**:

```json
{
  "code": 200,
  "message": "删除成功",
  "data": null
}
```

**错误响应示例**:

```json
{
  "code": 400,
  "message": "library_id不能为空",
  "data": null
}
```

---

## 5. 调试接口

### 5.1 获取单个项目的选择码预览

**接口**: `GET /api/emby/pickcode-preview`

**描述**: 调试接口，用于测试从 Emby PlaybackInfo 中提取选择码的功能

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| item_id | string | 是 | Emby 项目 ID |

**请求示例**:

```
GET /api/emby/pickcode-preview?item_id=65102
```

**响应示例 (成功)**:

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "pickcode": "a706hdog2st3zhfw6",
    "path": "http://qms.mqfamily.top:12333/115/url/video.mkv?pickcode=a706hdog2st3zhfw6"
  }
}
```

**错误响应示例**:

```json
{
  "code": 400,
  "message": "未从PlaybackInfo中解析到pickcode",
  "data": null
}
```

**说明**:

- 此接口用于调试和验证选择码提取逻辑
- 该接口的功能在同步时会自动执行
- 如果提取失败，需要检查 Emby 配置和网络连接

---

## 前端实现参考

### 1. 配置页面流程

```javascript
// 1. 获取当前配置
async function getEmbyConfig() {
  const response = await fetch('/api/setting/emby-config', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  return response.json();
}

// 2. 验证和更新配置
async function updateEmbyConfig(config) {
  const response = await fetch('/api/setting/emby-config', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(config)
  });
  const result = response.json();
  if (result.code === 200) {
    showSuccess('配置更新成功');
  } else {
    showError(result.message);
  }
}

// 3. 测试连接（通过获取同步状态）
async function testConnection() {
  const response = await fetch('/api/emby/sync/status', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const result = response.json();
  if (result.data.exists) {
    showSuccess('Emby 连接成功');
  } else {
    showError('Emby 配置不完整');
  }
}
```

### 2. 同步管理页面流程

```javascript
// 1. 启动同步
async function startSync() {
  disableSyncButton(); // 禁用按钮
  
  const response = await fetch('/api/emby/sync/start', {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const result = response.json();
  
  if (result.code === 200) {
    showSuccess('同步已启动');
    startPollingStatus(); // 开始轮询状态
  } else {
    showError(result.message);
    enableSyncButton();
  }
}

// 2. 轮询同步状态
async function startPollingStatus() {
  const interval = setInterval(async () => {
    const response = await fetch('/api/emby/sync/status', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    const result = response.json();
    
    if (result.code === 200) {
      updateLastSyncTime(result.data.last_sync_time);
      updateMediaCount(result.data.total_items);
      
      // 同步完成时停止轮询
      if (shouldStopPolling()) {
        clearInterval(interval);
        enableSyncButton();
        showSuccess('同步完成');
        refreshMediaList();
      }
    }
  }, 3000); // 每3秒轮询一次
}

// 3. 加载媒体列表
async function loadMediaList(page = 1, libraryId = null, type = null) {
  const params = new URLSearchParams({
    page: page,
    page_size: 50,
    library_id: libraryId || '',
    type: type || ''
  });
  
  const response = await fetch(`/api/emby/media?${params}`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const result = response.json();
  
  if (result.code === 200) {
    renderMediaTable(result.data.items);
    renderPagination(result.data.total, 50);
  }
}
```

### 3. 库关联管理流程

```javascript
// 1. 获取所有关联
async function loadRelations() {
  const response = await fetch('/api/emby/library-sync-paths', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const result = response.json();
  renderRelationsList(result.data);
}

// 2. 添加关联
async function addRelation(libraryId, syncPathId) {
  const response = await fetch('/api/emby/library-sync-paths', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      library_id: libraryId,
      sync_path_id: syncPathId
    })
  });
  
  const result = response.json();
  if (result.code === 200) {
    showSuccess('关联已保存');
    loadRelations();
  } else {
    showError(result.message);
  }
}

// 3. 删除关联
async function deleteRelation(libraryId) {
  if (!confirm('确定要删除此关联吗？')) return;
  
  const response = await fetch(`/api/emby/library-sync-paths?library_id=${libraryId}`, {
    method: 'DELETE',
    headers: { 'Authorization': `Bearer ${token}` }
  });
  
  const result = response.json();
  if (result.code === 200) {
    showSuccess('关联已删除');
    loadRelations();
  } else {
    showError(result.message);
  }
}
```

---

## 常见问题和故障排查

### Q1: 同步任务失败，提示 "已有Emby同步任务正在运行"

**原因**: 上一个同步任务仍在运行或未完成

**解决方案**:
1. 等待前一个任务完成（检查日志）
2. 应用重启后会自动清理运行状态
3. 检查 Emby 服务器网络连接

### Q2: 选择码提取失败

**原因**: 
- Emby 服务器无法访问 115 网盘资源
- PlaybackInfo 接口返回异常
- 网络超时

**解决方案**:
1. 检查 Emby 和 115 网盘的连接
2. 验证 Emby URL 和 API Key 是否正确
3. 查看服务器日志了解具体错误
4. 使用调试接口 `/api/emby/pickcode-preview` 测试

### Q3: 同步速度慢

**原因**:
- 网络连接不稳定
- Emby 服务器负载高
- 媒体库规模过大

**解决方案**:
1. 检查网络连接质量
2. 在非高峰时段执行同步
3. 分批同步不同的媒体库
4. 增加工作线程数（需要修改代码）

### Q4: 媒体查询无结果

**原因**:
- 从未执行过同步
- 同步失败
- 查询条件不匹配

**解决方案**:
1. 检查 last_sync_time，确认已执行过同步
2. 查看同步状态中的 total_items
3. 尝试不带筛选条件查询
4. 检查媒体库 ID 是否正确

### Q5: Cron 表达式无法识别

**原因**: 表达式格式错误或不支持的语法

**解决方案**:
1. 使用标准 cron 格式 `* * * * *`（5 个字段）
2. 避免使用非标准扩展（如 `@daily`）
3. 验证表达式的有效性
4. 参考文档中的常见示例

---

## 性能和限制

| 指标 | 值 |
|-----|-----|
| 最大并发同步数 | 1 |
| 同步工作线程数 | 10 |
| PlaybackInfo 请求超时 | 30 秒 |
| 媒体项目分页最大值 | 200 |
| 支持的媒体类型 | Movie, Series, Season, Episode, BoxSet 等 |
| 同步保留历史天数 | 取决于系统配置 |

---

## 更新日志

### v1.0.0 (2026-01-23)

- ✅ 初始版本
- ✅ 支持 Emby 配置管理
- ✅ 支持媒体同步和查询
- ✅ 支持库关联管理
- ✅ 支持选择码自动提取
- ✅ 支持定时同步（cron）

---

## 技术支持

如有问题或建议，请联系后端开发团队或提交 Issue。
