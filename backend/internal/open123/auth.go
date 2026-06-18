package open123

import (
	"encoding/json"
	"fmt"
	"time"
)

type TokenRequest struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiredAt   string `json:"expiredAt"`
}

type TokenResult struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    TokenResponse `json:"data"`
}

func (c *Client) performTokenRefresh() error {
	url := fmt.Sprintf("%s/api/v1/access_token", c.baseURL)

	req := TokenRequest{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal token request failed: %w", err)
	}

	resp, err := c.client.R().
		SetHeader("Platform", "open_platform").
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", c.ua).
		SetBody(body).
		Post(url)

	if err != nil {
		return fmt.Errorf("request token failed: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return fmt.Errorf("token request failed with status: %s", resp.Status())
	}

	result := &TokenResult{}
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return fmt.Errorf("unmarshal token response failed: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("token request failed: code=%d, message=%s", result.Code, result.Message)
	}

	expiredAt, err := time.Parse(time.RFC3339, result.Data.ExpiredAt)
	if err != nil {
		return fmt.Errorf("parse expired time failed: %w", err)
	}

	c.tokenMu.Lock()
	c.accessToken = result.Data.AccessToken
	c.expiredAt = expiredAt
	c.tokenMu.Unlock()

	return nil
}

func (c *Client) GetAccessToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.accessToken
}

func (c *Client) GetExpiredAt() time.Time {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.expiredAt
}
