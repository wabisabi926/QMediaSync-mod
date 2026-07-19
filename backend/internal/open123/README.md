# 123 云盘客户端当前接口

`backend/internal/open123` 是 123 云盘开放平台客户端的基础封装。当前项目主流程没有导入该包；仓库只包含包内单元测试，未接入真实账号、Token 或端到端调用。它不是当前可直接启用的 123 云盘来源实现。

## 当前限制

- `NewClient()` 只保存 `clientID`、`clientSecret`，并设置 30 秒 HTTP 超时；没有导出的 Token 设置方法。包外调用方无法为客户端写入 Access Token。
- `performTokenRefresh()` 已实现请求 Token 的逻辑，但 `refreshAccessToken()` 当前没有调用它。因此 Token 过期时不会真正刷新，不能把并发刷新框架视为可用的自动鉴权。
- `initDefaultRateLimits()` 是未导出的内部方法，且 `NewClient()` 不会调用它。即使包内调用，该方法配置的是路径前缀，而 `SetRateLimit(path, qps)` 和请求流程按完整 `URL.Path` 精确匹配；这些前缀不会限制普通接口请求。
- `UploadFile()` 的实际文件上传请求直接使用底层 HTTP 客户端，绕过 `doRequest()`；为具体 API 配置的限流和 Token 过期检查不会作用于这一步。
- 对外方法目前以 `fmt.Errorf` 返回 HTTP 和业务错误，不会构造 `APIError`。`IsTokenExpired`、`IsRateLimited` 不能直接判断这些方法返回的错误。

## 创建客户端

先通过 `open123.NewClient` 创建客户端，再按需要为每个实际请求路径调用 `SetRateLimit` 配置限流。使用完成后调用 `Close()` 关闭底层 HTTP 客户端。

以下示例仅说明当前构造和限流 API；在 Token 初始化和刷新流程补齐前，不能据此发起成功的真实 API 请求。

```go
client := open123.NewClient(clientID, clientSecret)
defer client.Close()

client.SetRateLimit("/api/v2/file/list", 10)
```

不要使用 `/api/v2/` 这类前缀代替实际路径；当前匹配不会按前缀生效。

## 当前可调用能力

### 客户端和速率限制

- `NewClient(clientID, clientSecret string) *Client`
- `Close() error`
- `SetRateLimit(path string, qps int)`
- `GetAccessToken() string`
- `GetExpiredAt() time.Time`

### 文件和目录

- `ListFiles(ctx, parentFileID, page, pageSize)`
- `CreateFolder(ctx, name, parentFileID)`
- `DeleteFile(ctx, fileID)`
- `DeleteFolder(ctx, dirID)`

`ListFiles` 返回目录内容，调用方自行遍历结果；`CreateFolder` 返回新建目录信息；`DeleteFile` 和 `DeleteFolder` 用于删除对象。

```go
files, err := client.ListFiles(ctx, 0, 1, 50)
```

### 上传

- `GetUploadDomain(ctx) (string, error)`
- `UploadFile(ctx, filePath, parentFileID)`

`UploadFile` 负责上传本地文件并返回上传结果，`GetUploadDomain` 用于获取上传域名。

```go
uploadResult, err := client.UploadFile(ctx, filePath, parentFileID)
```

### 下载链接

- `GetFileDownloadInfo(ctx, fileID)`
- `GetDirectLink(ctx, fileID)`

`GetFileDownloadInfo` 返回下载信息，`GetDirectLink` 返回直链地址。

```go
downloadInfo, err := client.GetFileDownloadInfo(ctx, fileID)
```

`types.go` 中的请求、响应和错误结构体也以导出形式存在，供上述方法的入参、返回值和 JSON 解码使用；它们不是当前项目对外维护的 123 云盘协议承诺，字段以源码和 123 云盘开放平台为准。

## 错误辅助

`APIError`、`NewAPIError`、`IsTokenExpired` 和 `IsRateLimited` 仅适用于调用方已经拿到或自行构造的 `*APIError`。当前 `ListFiles`、上传、下载等方法不会返回这种类型，不能对其返回值直接依赖这两个判断函数。

```go
if open123.IsTokenExpired(err) {
    return err
}
```

## 并发刷新框架

当前请求流程会在 Token 临近过期时进入 `ensureValidAccessToken()`，并通过 `sync.Once`、互斥锁和 channel 协调并发刷新，避免多个请求同时刷新。

但由于 `refreshAccessToken()` 尚未接入 `performTokenRefresh()`，这里只能视为并发刷新框架，不能视为完整的 Token 自动管理能力。

## 接入前需要补齐的边界

若要把本包接入实际同步流程，至少需要设计并验证：

- Access Token 的安全初始化、持久化边界和过期刷新；
- 所有请求（包括文件上传）的统一鉴权、限流和错误类型转换；
- 路径限流的精确匹配或前缀匹配规则；
- 真实 123 云盘账号的接口兼容性与端到端回归测试。
