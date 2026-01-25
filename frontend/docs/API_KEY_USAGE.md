# API Key 认证功能使用说明

## 功能概述

新增了 API Key 认证机制，允许通过 API Key 调用所有需要认证的接口，无需使用 JWT Token。

## 主要特性

- ✅ API Key 通过 GET 参数 `api_key` 传递
- ✅ API Key 使用 SHA256 哈希存储，安全可靠
- ✅ 每个 API Key 关联到创建它的用户
- ✅ API Key 验证成功后等同于 JWT Token 认证通过
- ✅ 自动记录 API Key 的最后使用时间
- ✅ 支持生成、查看列表、删除操作

## API 接口

### 1. 创建 API Key

**接口**: `POST /api/api-keys`

**需要认证**: 是（JWT Token）

**请求体**:

```json
{
  "name": "我的API Key"
}
```

**响应**:

```json
{
  "code": 200,
  "message": "API Key创建成功，请妥善保管密钥，此密钥仅显示一次",
  "data": {
    "id": 1,
    "name": "我的API Key",
    "key": "qms_a1B2c3D4e5F6g7H8i9J0k1L2",
    "key_prefix": "qms_a1B2",
    "created_at": 1737724800,
    "is_active": true
  }
}
```

⚠️ **重要**: 完整的 `key` 仅在创建时返回一次，请妥善保管！

### 2. 查看 API Key 列表

**接口**: `GET /api/api-keys`

**需要认证**: 是（JWT Token）

**响应**:

```json
{
  "code": 200,
  "message": "查询成功",
  "data": [
    {
      "id": 1,
      "name": "我的API Key",
      "key_prefix": "qms_a1B2",
      "last_used_at": 1737724850,
      "created_at": 1737724800,
      "is_active": true
    }
  ]
}
```

### 3. 更新 API Key 状态（启用/禁用）

**接口**: `PUT /api-keys/:id/status`

**需要认证**: 是（JWT Token）

**请求体**:

```json
{
  "is_active": false
}
```

**响应**:

```json
{
  "code": 200,
  "message": "API Key已禁用",
  "data": null
}
```

### 4. 删除 API Key

**接口**: `DELETE /api/api-keys/:id`

**需要认证**: 是（JWT Token）

**响应**:

```json
{
  "code": 200,
  "message": "删除成功",
  "data": null
}
```

## 使用方式

### 方式一：通过 GET 参数传递（推荐）

所有需要认证的接口都可以通过在 URL 中添加 `api_key` 参数来使用：

```bash
# 获取用户信息
GET /api/user/info?api_key=qms_a1B2c3D4e5F6g7H8i9J0k1L2

# 获取同步路径列表
GET /api/sync/path-list?api_key=qms_a1B2c3D4e5F6g7H8i9J0k1L2

# 启动同步
POST /api/sync/start?api_key=qms_a1B2c3D4e5F6g7H8i9J0k1L2
```

### 方式二：继续使用 JWT Token

如果 API Key 验证失败或未提供，系统会自动回退到 JWT Token 验证：

```bash
GET /api/user/info
Authorization: Bearer <your-jwt-token>
```

## 使用场景

- ✅ 第三方应用集成
- ✅ 自动化脚本调用
- ✅ Webhook 回调
- ✅ 定时任务
- ✅ API 测试和调试

## 管理功能

- ✅ **创建**: 为不同应用创建独立的 API Key
- ✅ **列表**: 查看所有 API Key 及其状态
- ✅ **禁用/启用**: 临时禁用或重新启用 API Key，无需删除
- ✅ **删除**: 永久删除不再使用的 API Key
- ✅ **追踪**: 通过 `last_used_at` 监控使用情况

## 安全建议

1. 🔐 妥善保管 API Key，不要泄露给他人
2. 📝 为不同的应用创建不同的 API Key，方便管理和追踪
3. ⏸️ 暂时不用的 API Key 可以禁用而不是删除，需要时可重新启用
4. 🗑️ 定期删除不再使用的 API Key
5. 📊 通过 `last_used_at` 字段监控 API Key 的使用情况
6. 🔒 如果 API Key 泄露，立即禁用或删除并创建新的

## 技术细节

- **密钥格式**: `qms_` + 24位随机字符（a-zA-Z0-9）
- **存储方式**: SHA256 哈希，数据库不存储明文
- **前缀显示**: 仅显示前8位（如 `qms_a1B2`），用于识别
- **验证优先级**: API Key > JWT Token
- **使用时间**: 每次使用自动更新 `last_used_at`

## 示例：使用 curl 测试

```bash
# 1. 先登录获取 JWT Token
curl -X POST http://localhost:12333/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 2. 创建 API Key
curl -X POST http://localhost:12333/api/api-keys \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"测试API Key"}'

# 3. 使用 API Key 调用接口
curl "http://localhost:12333/api/user/info?api_key=qms_a1B2c3D4e5F6g7H8i9J0k1L2"

# 4. 查看 API Key 列表
curl -X GET http://localhost:12333/api/api-keys \
  -H "Authorization: Bearer <your-jwt-token>"

# 5. 禁用 API Key
curl -X PUT http://localhost:12333/api/api-keys/1/status \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"is_active":false}'

# 6. 启用 API Key
curl -X PUT http://localhost:12333/api/api-keys/1/status \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"is_active":true}'

# 7. 删除 API Key
curl -X DELETE http://localhost:12333/api/api-keys/1 \
  -H "Authorization: Bearer <your-jwt-token>"
```

## 数据库表结构

创建的 `api_keys` 表结构：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 关联的用户ID |
| name | string | API Key名称 |
| key_hash | string | SHA256哈希值（唯一索引） |
| key_prefix | string | 前8位明文，用于显示 |
| last_used_at | int64 | 最后使用时间戳 |
| is_active | bool | 是否启用 |
| created_at | int64 | 创建时间 |
| updated_at | int64 | 更新时间 |
