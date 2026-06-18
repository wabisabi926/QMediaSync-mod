# 123云盘客户端使用示例

## 创建客户端

```go
import "Q115-STRM/internal/open123"

func main() {
    clientID := "your_client_id"
    clientSecret := "your_client_secret"

    client := open123.NewClient(clientID, clientSecret)
    defer client.Close()

    client.initDefaultRateLimits()
}
```

## 获取文件列表

```go
ctx := context.Background()
files, err := client.ListFiles(ctx, 0, 1, 50)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total files: %d\n", files.Total)
for _, file := range files.FileList {
    fmt.Printf("File: %s (ID: %d, Size: %d)\n", file.FileName, file.FileID, file.FileSize)
}
```

## 创建文件夹

```go
folder, err := client.CreateFolder(ctx, "MyFolder", 0)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created folder with ID: %d\n", folder.DirID)
```

## 上传文件

```go
filePath := "/path/to/local/file.txt"
parentID := int64(0)

uploadResult, err := client.UploadFile(ctx, filePath, parentID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Uploaded file with ID: %d\n", uploadResult.FileID)
```

## 获取下载链接

```go
fileID := int64(12345)

downloadInfo, err := client.GetFileDownloadInfo(ctx, fileID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Download URL: %s\n", downloadInfo.DownloadURL)
fmt.Printf("File Name: %s\n", downloadInfo.FileName)
fmt.Printf("File Size: %d\n", downloadInfo.FileSize)
fmt.Printf("Expire Time: %d\n", downloadInfo.ExpireTime)
```

## 删除文件

```go
fileID := int64(12345)

err := client.DeleteFile(ctx, fileID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Deleted file with ID: %d\n", fileID)
```

## 删除文件夹

```go
dirID := int64(12345)

err := client.DeleteFolder(ctx, dirID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Deleted folder with ID: %d\n", dirID)
```

## 配置速率限制

```go
client := open123.NewClient(clientID, clientSecret)

client.SetRateLimit("/api/v1/", 10)
client.SetRateLimit("/upload/v2/", 5)
client.SetRateLimit("/api/v2/", 15)
```

## 错误处理

```go
downloadInfo, err := client.GetFileDownloadInfo(ctx, fileID)
if err != nil {
    if open123.IsTokenExpired(err) {
        log.Println("Token expired, will be auto-refreshed on next request")
    } else if open123.IsRateLimited(err) {
        log.Println("Rate limit exceeded, please try again later")
    } else {
        log.Printf("Error: %v\n", err)
    }
}
```

## 并发安全

客户端自动处理并发情况下的token刷新：

- 多个请求同时检测到token过期时，只有一个请求执行刷新
- 其他请求等待刷新完成
- 使用sync.Once确保单一刷新操作
- 线程安全的token读写访问

## Token自动管理

- Token会在过期前30秒自动刷新
- 所有API调用都包含有效性检查
- 支持上下文取消和超时控制
