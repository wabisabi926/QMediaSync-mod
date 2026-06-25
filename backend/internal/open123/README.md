# 123 云盘客户端当前接口

`backend/internal/open123` 目前是 123 云盘开放平台客户端的基础封装。当前项目主流程尚未引用该包，仅 `open123` 包内测试覆盖客户端初始化、速率限制和部分并发逻辑。

## 当前限制

- `initDefaultRateLimits()` 是未导出的内部方法，其他包不能直接调用。
- `performTokenRefresh()` 已实现请求 Token 的逻辑，但 `refreshAccessToken()` 当前还没有调用它。
- 当前没有导出的 Token 设置方法，因此真实 API 请求前还需要补齐 Token 初始化/刷新流程。
- `SetRateLimit(path, qps)` 当前按完整 path 匹配，不按前缀匹配。

## 创建客户端

先通过 `open123.NewClient` 创建客户端，再按需要调用 `SetRateLimit` 配置各接口限流。使用完成后调用 `Close()` 关闭底层 HTTP 客户端。

```go
client := open123.NewClient(clientID, clientSecret)
defer client.Close()

client.SetRateLimit("/api/v2/file/list", 10)
```

## 已导出的能力

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

## 错误辅助

可配合 `IsTokenExpired` 和 `IsRateLimited` 判断常见错误类型，再按业务需要继续处理。

```go
if open123.IsTokenExpired(err) {
    return err
}
```

## 并发刷新框架

当前请求流程会在 Token 临近过期时进入 `ensureValidAccessToken()`，并通过 `sync.Once`、互斥锁和 channel 协调并发刷新，避免多个请求同时刷新。

但由于 `refreshAccessToken()` 尚未接入真实刷新动作，这里只能视为并发刷新框架，不能视为完整的 Token 自动管理能力。
