package open123

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

func (c *Client) GetUploadDomain(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/upload/v2/file/domain", c.baseURL)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return "", fmt.Errorf("get upload domain failed with status: %s", resp.Status())
	}

	result := &RespBase[[]string]{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return "", fmt.Errorf("unmarshal upload domain response failed: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("get upload domain failed: code=%d, message=%s", result.Code, result.Message)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("no upload domain available")
	}

	return result.Data[0], nil
}

func (c *Client) UploadFile(ctx context.Context, filePath string, parentFileID int64) (*FileUploadCreateResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("get file info failed: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file failed: %w", err)
	}
	defer file.Close()

	filename := filepath.Base(filePath)

	domain, err := c.GetUploadDomain(ctx)
	if err != nil {
		return nil, fmt.Errorf("get upload domain failed: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/upload/v2/file/single/create", domain)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("parentFileID", fmt.Sprintf("%d", parentFileID))
	_ = writer.WriteField("filename", filename)
	_ = writer.WriteField("size", fmt.Sprintf("%d", fileInfo.Size()))

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file failed: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy file to form failed: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer failed: %w", err)
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.GetAccessToken()).
		SetHeader("Platform", "open_platform").
		SetHeader("User-Agent", c.ua).
		SetBody(body.Bytes()).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Post(uploadURL)

	if err != nil {
		return nil, fmt.Errorf("upload file request failed: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("upload file failed with status: %s", resp.Status())
	}

	result := &RespBase[FileUploadCreateResponse]{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return nil, fmt.Errorf("unmarshal upload response failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("upload file failed: code=%d, message=%s", result.Code, result.Message)
	}

	return &result.Data, nil
}
