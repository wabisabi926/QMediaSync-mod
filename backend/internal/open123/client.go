package open123

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"resty.dev/v3"
)

type Client struct {
	clientID     string
	clientSecret string
	accessToken  string
	expiredAt    time.Time
	baseURL      string
	ua           string
	client       *resty.Client

	tokenMu          sync.RWMutex
	isRefreshing     sync.Mutex
	refreshOnce      sync.Once
	refreshTokenChan chan struct{}

	limiterLock sync.RWMutex
	limiters    map[string]*rate.Limiter
}

func NewClient(clientID, clientSecret string) *Client {
	client := resty.New()
	client.SetTimeout(time.Duration(DEFAULT_TIMEOUT) * time.Second)

	return &Client{
		clientID:         clientID,
		clientSecret:     clientSecret,
		baseURL:          OPEN_BASE_URL,
		ua:               DEFAULTUA,
		client:           client,
		refreshTokenChan: make(chan struct{}),
		limiters:         make(map[string]*rate.Limiter),
	}
}

func (c *Client) initDefaultRateLimits() {
	c.SetRateLimit("/api/v1/", 10)
	c.SetRateLimit("/upload/v2/", 5)
	c.SetRateLimit("/api/v2/", 15)
}

func (c *Client) SetRateLimit(path string, qps int) {
	c.limiterLock.Lock()
	defer c.limiterLock.Unlock()

	c.limiters[path] = rate.NewLimiter(rate.Limit(qps), 1)
}

func (c *Client) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

func (c *Client) waitForPermission(ctx context.Context, path string) error {
	c.limiterLock.RLock()
	limiter, exists := c.limiters[path]
	c.limiterLock.RUnlock()

	if exists {
		return limiter.Wait(ctx)
	}
	return nil
}

func (c *Client) doRequest(ctx context.Context, method, requestURL string, body []byte) (*resty.Response, error) {
	parsedURL, err := url.Parse(requestURL)
	var pathKey string
	if err != nil {
		pathKey = requestURL
	} else {
		pathKey = parsedURL.Path
	}

	if err := c.waitForPermission(ctx, pathKey); err != nil {
		return nil, fmt.Errorf("rate limit wait error: %w", err)
	}

	c.tokenMu.RLock()
	accessToken := c.accessToken
	isExpired := c.isTokenExpiredLocked()
	c.tokenMu.RUnlock()

	if isExpired {
		if err := c.ensureValidAccessToken(ctx); err != nil {
			return nil, err
		}
		c.tokenMu.RLock()
		accessToken = c.accessToken
		c.tokenMu.RUnlock()
	}

	req := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		SetHeader("Platform", "open_platform").
		SetHeader("User-Agent", c.ua)

	if body != nil {
		req = req.SetBody(body)
	}

	return req.Execute(method, requestURL)
}

func (c *Client) isTokenExpired() bool {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.isTokenExpiredLocked()
}

func (c *Client) isTokenExpiredLocked() bool {
	return time.Until(c.expiredAt) <= 30*time.Second
}

func (c *Client) ensureValidAccessToken(ctx context.Context) error {
	var refreshErr error
	c.refreshOnce.Do(func() {
		refreshErr = c.refreshAccessToken()
		if refreshErr != nil {
			c.refreshOnce = sync.Once{}
		} else {
			close(c.refreshTokenChan)
		}
	})

	if refreshErr != nil {
		return refreshErr
	}

	select {
	case <-c.refreshTokenChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("token refresh timeout")
	}
}

func (c *Client) refreshAccessToken() error {
	c.isRefreshing.Lock()
	defer c.isRefreshing.Unlock()

	c.tokenMu.RLock()
	if !c.isTokenExpiredLocked() {
		c.tokenMu.RUnlock()
		return nil
	}
	c.tokenMu.RUnlock()

	return nil
}
