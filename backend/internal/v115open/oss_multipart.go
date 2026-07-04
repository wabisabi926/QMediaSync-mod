package v115open

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

const (
	defaultMultipartPartSize int64 = 32 * 1024 * 1024
	multipartPartAlign       int64 = 1024 * 1024
	maxMultipartParts        int64 = 9999
	maxMultipartPartSize     int64 = 5 * 1024 * 1024 * 1024
)

type ossMultipartClient interface {
	InitiateMultipartUpload(context.Context, *oss.InitiateMultipartUploadRequest, ...func(*oss.Options)) (*oss.InitiateMultipartUploadResult, error)
	UploadPart(context.Context, *oss.UploadPartRequest, ...func(*oss.Options)) (*oss.UploadPartResult, error)
	ListParts(context.Context, *oss.ListPartsRequest, ...func(*oss.Options)) (*oss.ListPartsResult, error)
	CompleteMultipartUpload(context.Context, *oss.CompleteMultipartUploadRequest, ...func(*oss.Options)) (*oss.CompleteMultipartUploadResult, error)
	AbortMultipartUpload(context.Context, *oss.AbortMultipartUploadRequest, ...func(*oss.Options)) (*oss.AbortMultipartUploadResult, error)
}

// OSSMultipartUploader 封装 OSS multipart 上传。
type OSSMultipartUploader struct {
	client ossMultipartClient
}

// OSSMultipartUploadInput 是 multipart 上传输入。
type OSSMultipartUploadInput struct {
	Bucket        string
	Object        string
	Callback      string
	CallbackVar   string
	FilePath      string
	FileSize      int64
	FileSha1      string
	UploadId      string
	PartSize      int64
	PartRetryMax  int
	OnProgress    func(OSSMultipartProgress)
	refreshClient func(context.Context) (ossMultipartClient, error)
}

// OSSMultipartUploadedPart 是已上传分片。
type OSSMultipartUploadedPart struct {
	PartNumber int32
	ETag       string
	Size       int64
}

// OSSMultipartUploadResult 是 multipart 上传后的完整结果。
type OSSMultipartUploadResult struct {
	CallbackResult map[string]any
	UploadId       string
	PartSize       int64
	TotalParts     int
	UploadedBytes  int64
	UploadedParts  int
}

// OSSMultipartProgress 是 multipart 上传过程中的 checkpoint 进度。
type OSSMultipartProgress struct {
	UploadId       string
	PartSize       int64
	TotalParts     int
	UploadedBytes  int64
	UploadedParts  int
	LastPartNumber int
	LastPartEtag   string
}

// CalculateMultipartPartSize 计算 OSS multipart 分片大小和分片数量。
func CalculateMultipartPartSize(fileSize int64) (int64, int, error) {
	if fileSize < 0 {
		return 0, 0, fmt.Errorf("文件大小不能为负数：%d", fileSize)
	}
	partSize := defaultMultipartPartSize
	minPartSize := ceilDiv(fileSize, maxMultipartParts)
	if minPartSize > partSize {
		partSize = roundUp(minPartSize, multipartPartAlign)
	}
	if partSize > maxMultipartPartSize {
		return 0, 0, fmt.Errorf("文件过大，所需分片大小 %d 超过 OSS 上限 %d", partSize, maxMultipartPartSize)
	}
	totalParts := int(ceilDiv(fileSize, partSize))
	if totalParts == 0 {
		totalParts = 1
	}
	if int64(totalParts) > maxMultipartParts {
		return 0, 0, fmt.Errorf("分片数量 %d 超过上限 %d", totalParts, maxMultipartParts)
	}
	return partSize, totalParts, nil
}

// NewOSSMultipartUploader 创建 OSS multipart 上传器。
func NewOSSMultipartUploader(endpoint string, accessKeyId string, accessKeySecret string, securityToken string) *OSSMultipartUploader {
	return &OSSMultipartUploader{client: newOSSMultipartClient(endpoint, accessKeyId, accessKeySecret, securityToken)}
}

func newOSSMultipartClient(endpoint string, accessKeyId string, accessKeySecret string, securityToken string) ossMultipartClient {
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, securityToken)).
		WithRegion("cn-shenzhen").
		WithEndpoint(endpoint)
	return oss.NewClient(cfg)
}

func newOSSMultipartClientFromToken(token *UploadToken) (ossMultipartClient, error) {
	if token == nil {
		return nil, fmt.Errorf("上传凭证为空")
	}
	token.normalize()
	return newOSSMultipartClient(token.Endpoint, token.AccessKeyId, token.AccessKeySecret, token.SecurityToken), nil
}

// UploadFile 上传文件并完成 OSS multipart。
func (u *OSSMultipartUploader) UploadFile(ctx context.Context, input OSSMultipartUploadInput) (map[string]any, error) {
	result, err := u.UploadFileWithResult(ctx, input)
	if err != nil {
		return nil, err
	}
	return result.CallbackResult, nil
}

// UploadFileWithResult 上传文件并返回 multipart checkpoint 结果。
func (u *OSSMultipartUploader) UploadFileWithResult(ctx context.Context, input OSSMultipartUploadInput) (OSSMultipartUploadResult, error) {
	if input.PartRetryMax <= 0 {
		input.PartRetryMax = 3
	}
	partSize := input.PartSize
	totalParts := 0
	var err error
	if partSize <= 0 {
		partSize, totalParts, err = CalculateMultipartPartSize(input.FileSize)
		if err != nil {
			return OSSMultipartUploadResult{}, err
		}
	} else {
		totalParts = int(ceilDiv(input.FileSize, partSize))
	}

	uploadId := input.UploadId
	if uploadId == "" {
		initResult, err := u.client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
			Bucket: oss.Ptr(input.Bucket),
			Key:    oss.Ptr(input.Object),
		})
		if err != nil {
			return OSSMultipartUploadResult{}, fmt.Errorf("初始化 OSS multipart 失败：%w", err)
		}
		if initResult.UploadId == nil || *initResult.UploadId == "" {
			return OSSMultipartUploadResult{}, fmt.Errorf("初始化 OSS multipart 返回空 upload_id")
		}
		uploadId = *initResult.UploadId
	}
	reportOSSMultipartProgress(input, OSSMultipartProgress{
		UploadId:   uploadId,
		PartSize:   partSize,
		TotalParts: totalParts,
	})

	existingParts, err := u.ListUploadedParts(ctx, input.Bucket, input.Object, uploadId)
	if err != nil {
		return OSSMultipartUploadResult{}, err
	}
	existingPartMap := make(map[int32]OSSMultipartUploadedPart, len(existingParts))
	for _, part := range existingParts {
		existingPartMap[part.PartNumber] = part
	}

	file, err := os.Open(input.FilePath)
	if err != nil {
		return OSSMultipartUploadResult{}, fmt.Errorf("打开待上传文件失败：%w", err)
	}
	defer file.Close()

	var uploadedBytes int64
	uploadedParts := 0
	completeParts := make([]oss.UploadPart, 0, totalParts)
	for partNumber := 1; partNumber <= totalParts; partNumber++ {
		offset := int64(partNumber-1) * partSize
		length := minInt64(partSize, input.FileSize-offset)
		if length < 0 {
			length = 0
		}
		if existing, ok := existingPartMap[int32(partNumber)]; ok && existing.Size == length && existing.ETag != "" {
			uploadedBytes += existing.Size
			uploadedParts++
			completeParts = append(completeParts, oss.UploadPart{
				PartNumber: int32(partNumber),
				ETag:       oss.Ptr(existing.ETag),
			})
			reportOSSMultipartProgress(input, OSSMultipartProgress{
				UploadId:       uploadId,
				PartSize:       partSize,
				TotalParts:     totalParts,
				UploadedBytes:  uploadedBytes,
				UploadedParts:  uploadedParts,
				LastPartNumber: partNumber,
				LastPartEtag:   existing.ETag,
			})
			continue
		}

		etag, err := u.uploadPartWithRetry(ctx, input, uploadId, int32(partNumber), file, offset, length)
		if err != nil {
			return OSSMultipartUploadResult{}, err
		}
		uploadedBytes += length
		uploadedParts++
		completeParts = append(completeParts, oss.UploadPart{
			PartNumber: int32(partNumber),
			ETag:       oss.Ptr(etag),
		})
		reportOSSMultipartProgress(input, OSSMultipartProgress{
			UploadId:       uploadId,
			PartSize:       partSize,
			TotalParts:     totalParts,
			UploadedBytes:  uploadedBytes,
			UploadedParts:  uploadedParts,
			LastPartNumber: partNumber,
			LastPartEtag:   etag,
		})
	}
	sort.Slice(completeParts, func(i, j int) bool {
		return completeParts[i].PartNumber < completeParts[j].PartNumber
	})

	headers, err := BuildOSSCallbackHeaders(OSSCallbackInput{
		Callback:    input.Callback,
		CallbackVar: input.CallbackVar,
		Bucket:      input.Bucket,
		Object:      input.Object,
		FileSize:    input.FileSize,
		FileSha1:    input.FileSha1,
	})
	if err != nil {
		return OSSMultipartUploadResult{}, err
	}
	completeResult, err := u.client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(input.Bucket),
		Key:      oss.Ptr(input.Object),
		UploadId: oss.Ptr(uploadId),
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: completeParts,
		},
		Callback:    oss.Ptr(headers.Callback),
		CallbackVar: oss.Ptr(headers.CallbackVar),
	})
	if err != nil {
		return OSSMultipartUploadResult{}, fmt.Errorf("完成 OSS multipart 失败：%w", err)
	}
	return OSSMultipartUploadResult{
		CallbackResult: completeResult.CallbackResult,
		UploadId:       uploadId,
		PartSize:       partSize,
		TotalParts:     totalParts,
		UploadedBytes:  uploadedBytes,
		UploadedParts:  uploadedParts,
	}, nil
}

func reportOSSMultipartProgress(input OSSMultipartUploadInput, progress OSSMultipartProgress) {
	if input.OnProgress == nil {
		return
	}
	input.OnProgress(progress)
}

// ListUploadedParts 查询 OSS 已上传分片。
func (u *OSSMultipartUploader) ListUploadedParts(ctx context.Context, bucket string, object string, uploadId string) ([]OSSMultipartUploadedPart, error) {
	parts := []OSSMultipartUploadedPart{}
	var marker int32
	for {
		result, err := u.client.ListParts(ctx, &oss.ListPartsRequest{
			Bucket:           oss.Ptr(bucket),
			Key:              oss.Ptr(object),
			UploadId:         oss.Ptr(uploadId),
			MaxParts:         1000,
			PartNumberMarker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("查询 OSS 已上传分片失败：%w", err)
		}
		for _, part := range result.Parts {
			etag := ""
			if part.ETag != nil {
				etag = *part.ETag
			}
			parts = append(parts, OSSMultipartUploadedPart{
				PartNumber: part.PartNumber,
				ETag:       etag,
				Size:       part.Size,
			})
		}
		if !result.IsTruncated {
			break
		}
		marker = result.NextPartNumberMarker
	}
	return parts, nil
}

// Abort 取消 OSS multipart 上传。
func (u *OSSMultipartUploader) Abort(ctx context.Context, bucket string, object string, uploadId string) error {
	_, err := u.client.AbortMultipartUpload(ctx, &oss.AbortMultipartUploadRequest{
		Bucket:   oss.Ptr(bucket),
		Key:      oss.Ptr(object),
		UploadId: oss.Ptr(uploadId),
	})
	return err
}

func (u *OSSMultipartUploader) uploadPartWithRetry(
	ctx context.Context,
	input OSSMultipartUploadInput,
	uploadId string,
	partNumber int32,
	file *os.File,
	offset int64,
	length int64,
) (string, error) {
	var lastErr error
	for attempt := 0; attempt < input.PartRetryMax; attempt++ {
		reader := io.NewSectionReader(file, offset, length)
		result, err := u.client.UploadPart(ctx, &oss.UploadPartRequest{
			Bucket:        oss.Ptr(input.Bucket),
			Key:           oss.Ptr(input.Object),
			PartNumber:    partNumber,
			UploadId:      oss.Ptr(uploadId),
			Body:          reader,
			ContentLength: oss.Ptr(length),
		})
		if err == nil {
			if result.ETag == nil || *result.ETag == "" {
				return "", fmt.Errorf("OSS part %d 返回空 ETag", partNumber)
			}
			return *result.ETag, nil
		}
		lastErr = err
		if attempt < input.PartRetryMax-1 && input.refreshClient != nil {
			refreshedClient, refreshErr := input.refreshClient(ctx)
			if refreshErr != nil {
				lastErr = refreshErr
				continue
			}
			u.client = refreshedClient
		}
	}
	return "", fmt.Errorf("上传 OSS part %d 失败：%w", partNumber, lastErr)
}

func ceilDiv(n int64, d int64) int64 {
	if d <= 0 {
		return 0
	}
	if n <= 0 {
		return 0
	}
	return (n + d - 1) / d
}

func roundUp(n int64, align int64) int64 {
	if align <= 0 {
		return n
	}
	return ceilDiv(n, align) * align
}

func minInt64(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// OssUploadFile 是旧上传入口的兼容包装，内部使用 OSS multipart。
func OssUploadFile(
	endPoint string,
	accessKeyId string,
	accessKeySecret string,
	securityToken string,
	bucketName string,
	objectId string,
	callback string,
	callbackVar string,
	filePath string,
	fileSize int64,
	fileSha1 string,
) (map[string]any, error) {
	uploader := NewOSSMultipartUploader(endPoint, accessKeyId, accessKeySecret, securityToken)
	return uploader.UploadFile(context.Background(), OSSMultipartUploadInput{
		Bucket:      bucketName,
		Object:      objectId,
		Callback:    callback,
		CallbackVar: callbackVar,
		FilePath:    filePath,
		FileSize:    fileSize,
		FileSha1:    fileSha1,
	})
}
