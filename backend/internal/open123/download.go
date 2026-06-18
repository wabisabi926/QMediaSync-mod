package open123

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) GetFileDownloadInfo(ctx context.Context, fileID int64) (*FileDownloadInfoResponse, error) {
	url := fmt.Sprintf("%s/api/v1/file/download_info", c.baseURL)

	req := FileDownloadInfoRequest{
		FileID:  fileID,
		DriveID: 0,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal download info request failed: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get download info failed with status: %s", resp.Status())
	}

	result := &RespBase[FileDownloadInfoResponse]{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return nil, fmt.Errorf("unmarshal download info response failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("get download info failed: code=%d, message=%s", result.Code, result.Message)
	}

	return &result.Data, nil
}

func (c *Client) GetDirectLink(ctx context.Context, fileID int64) (string, error) {
	info, err := c.GetFileDownloadInfo(ctx, fileID)
	if err != nil {
		return "", err
	}

	return info.DownloadURL, nil
}
