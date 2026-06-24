# 123云盘客户端当前接口

`backend/internal/open123` 目前是 123 云盘开放平台客户端的基础封装。当前项目主流程尚未引用该包，仅 `open123` 包内测试覆盖客户端初始化、速率限制和部分并发逻辑。

## 当前限制

- `initDefaultRateLimits()` 是未导出的内部方法，其他包不能直接调用。
- `performTokenRefresh()` 已实现请求 token 的逻辑，但 `refreshAccessToken()` 当前还没有调用它。
- 当前没有导出的 token 设置方法，因此真实 API 请求前还需要补齐 token 初始化/刷新流程。

## 创建客户端

```go
import "Q115-STRM/internal/open123"

func newOpen123Client() *open123.Client {
    client := open123.NewClient("your_client_id", "your_client_secret")

    client.SetRateLimit("/api/v1/", 10)
    client.SetRateLimit("/upload/v2/", 5)
    client.SetRateLimit("/api/v2/", 15)

    return client
}
```

使用完成后关闭底层 HTTP 客户端：

```go
defer client.Close()
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

```go
ctx := context.Background()

files, err := client.ListFiles(ctx, 0, 1, 50)
if err != nil {
    log.Fatal(err)
}

for _, file := range files.FileList {
    fmt.Printf("File: %s (ID: %d, Size: %d)\n", file.FileName, file.FileID, file.FileSize)
}
```

```go
folder, err := client.CreateFolder(ctx, "MyFolder", 0)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created folder with ID: %d\n", folder.DirID)
```

### 上传

- `GetUploadDomain(ctx) (string, error)`
- `UploadFile(ctx, filePath, parentFileID)`

```go
uploadResult, err := client.UploadFile(ctx, "/path/to/local/file.txt", 0)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Uploaded file with ID: %d\n", uploadResult.FileID)
```

### 下载链接

- `GetFileDownloadInfo(ctx, fileID)`
- `GetDirectLink(ctx, fileID)`

```go
downloadInfo, err := client.GetFileDownloadInfo(ctx, 12345)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Download URL: %s\n", downloadInfo.DownloadURL)
fmt.Printf("File Name: %s\n", downloadInfo.FileName)
fmt.Printf("File Size: %d\n", downloadInfo.FileSize)
fmt.Printf("Expire Time: %d\n", downloadInfo.ExpireTime)
```

## 错误辅助

```go
downloadInfo, err := client.GetFileDownloadInfo(ctx, fileID)
if err != nil {
    if open123.IsTokenExpired(err) {
        log.Println("token expired")
    } else if open123.IsRateLimited(err) {
        log.Println("rate limit exceeded")
    } else {
        log.Printf("error: %v\n", err)
    }
}
```

## 并发刷新框架

当前请求流程会在 token 临近过期时进入 `ensureValidAccessToken()`，并通过 `sync.Once`、互斥锁和 channel 协调并发刷新，避免多个请求同时刷新。

但由于 `refreshAccessToken()` 尚未接入真实刷新动作，这里只能视为并发刷新框架，不能视为完整的 token 自动管理能力。
