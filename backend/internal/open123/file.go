package open123

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) ListFiles(ctx context.Context, parentFileID int64, page, pageSize int) (*FileListResponse, error) {
	url := fmt.Sprintf("%s/api/v2/file/list?driveID=0&parentFileID=%d&page=%d&pageSize=%d", c.baseURL, parentFileID, page, pageSize)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("list files failed with status: %s", resp.Status())
	}

	result := &RespBase[FileListResponse]{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return nil, fmt.Errorf("unmarshal list files response failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("list files failed: code=%d, message=%s", result.Code, result.Message)
	}

	return &result.Data, nil
}

func (c *Client) CreateFolder(ctx context.Context, name string, parentFileID int64) (*CreateFolderResponse, error) {
	url := fmt.Sprintf("%s/api/v1/dir/create", c.baseURL)

	req := CreateFolderRequest{
		DriveID:      0,
		ParentFileID: parentFileID,
		DirName:      name,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal create folder request failed: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("create folder failed with status: %s", resp.Status())
	}

	result := &RespBase[CreateFolderResponse]{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return nil, fmt.Errorf("unmarshal create folder response failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("create folder failed: code=%d, message=%s", result.Code, result.Message)
	}

	return &result.Data, nil
}

func (c *Client) DeleteFile(ctx context.Context, fileID int64) error {
	url := fmt.Sprintf("%s/api/v1/file/delete", c.baseURL)

	req := DeleteRequest{
		FileID:  fileID,
		DriveID: 0,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal delete file request failed: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return fmt.Errorf("delete file failed with status: %s", resp.Status())
	}

	result := &RespBase[any]{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return fmt.Errorf("unmarshal delete file response failed: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("delete file failed: code=%d, message=%s", result.Code, result.Message)
	}

	return nil
}

func (c *Client) DeleteFolder(ctx context.Context, dirID int64) error {
	url := fmt.Sprintf("%s/api/v1/dir/delete", c.baseURL)

	req := DeleteFolderRequest{
		DirID:   dirID,
		DriveID: 0,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal delete folder request failed: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return fmt.Errorf("delete folder failed with status: %s", resp.Status())
	}

	result := &RespBase[any]{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return fmt.Errorf("unmarshal delete folder response failed: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("delete folder failed: code=%d, message=%s", result.Code, result.Message)
	}

	return nil
}
