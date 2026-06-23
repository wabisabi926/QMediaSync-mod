package v115auth

import (
	"Q115-STRM/internal/helpers"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type OAuthURLRequest struct {
	AccountID   uint
	AppID       string
	RedirectURL string
	Provider    AuthProvider
}

type OAuthURLResult struct {
	AuthURL   string `json:"auth_url,omitempty"`
	State     string `json:"state,omitempty"`
	Polling   bool   `json:"polling"`
	ExpiresIn int64  `json:"expires_in,omitempty"`
}

type OAuthTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	Done         bool
}

type OAuthProvider interface {
	BuildAuth(ctx context.Context, req OAuthURLRequest) (OAuthURLResult, error)
	Confirm(ctx context.Context, payload map[string]string) (OAuthTokenResult, error)
	Poll(ctx context.Context, state string) (OAuthTokenResult, error)
}

var errUnsupportedOAuthOperation = errors.New("当前授权服务不支持此操作")

func GetOAuthProvider(provider AuthProvider) (OAuthProvider, bool) {
	switch provider {
	case ProviderQMediaSync, ProviderMQFamily:
		return relayOAuthProvider{}, true
	case ProviderMoviePilot:
		return moviePilotOAuthProvider{authServer: "https://movie-pilot.org", client: defaultOAuthHTTPClient()}, true
	case ProviderOpenList:
		return openListOAuthProvider{client: defaultOAuthHTTPClient()}, true
	case ProviderCloudDrive:
		return cloudDriveOAuthProvider{}, true
	default:
		return nil, false
	}
}

func defaultOAuthHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

type relayOAuthProvider struct{}

func (provider relayOAuthProvider) BuildAuth(_ context.Context, req OAuthURLRequest) (OAuthURLResult, error) {
	clientID := strings.TrimSpace(req.AppID)
	if clientID == "" {
		clientID = BuiltInRelayQ115STRM
		if req.Provider == ProviderQMediaSync {
			clientID = BuiltInRelayQMediaSync
		}
	}
	redirectURL := strings.TrimSpace(req.RedirectURL)
	if redirectURL != "" {
		redirectURL = fmt.Sprintf("%s?source=115", redirectURL)
	}
	stateObj := struct {
		State       string `json:"state"`
		Time        int64  `json:"time"`
		ClientId    string `json:"client_id"`
		RedirectUrl string `json:"redirect_url"`
		AccountId   uint   `json:"account_id"`
	}{
		State:       helpers.RandStr(16),
		Time:        time.Now().Unix(),
		ClientId:    clientID,
		RedirectUrl: redirectURL,
		AccountId:   req.AccountID,
	}
	stateJSON, _ := json.Marshal(stateObj)
	stateEncoded, err := helpers.Encrypt(string(stateJSON))
	if err != nil {
		return OAuthURLResult{}, err
	}
	baseURL := helpers.GlobalConfig.AuthServer
	if req.Provider == ProviderQMediaSync || clientID == BuiltInRelayQMediaSync {
		baseURL = helpers.GlobalConfig.NewAuthServer
	}
	return OAuthURLResult{AuthURL: fmt.Sprintf("%s/115.php?action=code&state=%s", strings.TrimRight(baseURL, "/"), stateEncoded)}, nil
}

func (provider relayOAuthProvider) Confirm(_ context.Context, payload map[string]string) (OAuthTokenResult, error) {
	data := payload["data"]
	if data == "" {
		return OAuthTokenResult{}, fmt.Errorf("缺少中转回调数据")
	}
	decryptedData, err := helpers.Decrypt(data)
	if err != nil {
		return OAuthTokenResult{}, err
	}
	var resp struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(decryptedData), &resp); err != nil {
		return OAuthTokenResult{}, err
	}
	if resp.Data.AccessToken == "" || resp.Data.RefreshToken == "" {
		if resp.Error != "" {
			return OAuthTokenResult{}, errors.New(resp.Error)
		}
		if resp.Message != "" {
			return OAuthTokenResult{}, errors.New(resp.Message)
		}
		return OAuthTokenResult{}, fmt.Errorf("中转回调未返回访问凭证")
	}
	return OAuthTokenResult{AccessToken: resp.Data.AccessToken, RefreshToken: resp.Data.RefreshToken, ExpiresIn: resp.Data.ExpiresIn, Done: true}, nil
}

func (provider relayOAuthProvider) Poll(_ context.Context, _ string) (OAuthTokenResult, error) {
	return OAuthTokenResult{}, errUnsupportedOAuthOperation
}

type moviePilotOAuthProvider struct {
	authServer string
	client     *http.Client
}

func (provider moviePilotOAuthProvider) BuildAuth(ctx context.Context, req OAuthURLRequest) (OAuthURLResult, error) {
	endpoint := strings.TrimRight(provider.authServer, "/") + "/u115/auth_url"
	resp, err := httpGetJSON(ctx, provider.client, endpoint)
	if err != nil {
		return OAuthURLResult{}, err
	}
	authURL := stringField(resp, "auth_url")
	state := stringField(resp, "state")
	if authURL == "" || state == "" {
		return OAuthURLResult{}, fmt.Errorf("MoviePilot 授权服务响应缺少 auth_url 或 state")
	}
	SaveOAuthState(OAuthState{State: state, AccountID: req.AccountID, Provider: ProviderMoviePilot, RedirectURL: req.RedirectURL})
	return OAuthURLResult{AuthURL: authURL, State: state, Polling: true, ExpiresIn: OAuthStateTTLSeconds}, nil
}

func (provider moviePilotOAuthProvider) Confirm(_ context.Context, _ map[string]string) (OAuthTokenResult, error) {
	return OAuthTokenResult{}, errUnsupportedOAuthOperation
}

func (provider moviePilotOAuthProvider) Poll(ctx context.Context, state string) (OAuthTokenResult, error) {
	if _, ok := GetOAuthState(state, ProviderMoviePilot); !ok {
		return OAuthTokenResult{}, fmt.Errorf("授权状态不存在或已过期")
	}
	endpoint := strings.TrimRight(provider.authServer, "/") + "/u115/token?state=" + url.QueryEscape(state)
	resp, err := httpGetJSON(ctx, provider.client, endpoint)
	if err != nil {
		return OAuthTokenResult{}, err
	}
	token := tokenResultFromMap(resp)
	if !token.Done {
		return OAuthTokenResult{Done: false}, nil
	}
	DeleteOAuthState(state)
	return token, nil
}

type openListOAuthProvider struct {
	client *http.Client
}

func (provider openListOAuthProvider) BuildAuth(_ context.Context, req OAuthURLRequest) (OAuthURLResult, error) {
	state := helpers.RandStr(16)
	SaveOAuthState(OAuthState{State: state, AccountID: req.AccountID, Provider: ProviderOpenList, RedirectURL: req.RedirectURL})
	return OAuthURLResult{
		AuthURL:   "https://api.oplist.org/115cloud/requests?driver_txt=115cloud_go&server_use=true",
		State:     state,
		ExpiresIn: OAuthStateTTLSeconds,
	}, nil
}

func (provider openListOAuthProvider) Confirm(_ context.Context, payload map[string]string) (OAuthTokenResult, error) {
	state := payload["state"]
	if state != "" {
		if _, ok := GetOAuthState(state, ProviderOpenList); !ok {
			return OAuthTokenResult{}, fmt.Errorf("授权状态不存在或已过期")
		}
		defer DeleteOAuthState(state)
	}
	if token := tokenResultFromPayload(payload); token.Done {
		return token, nil
	}
	encoded := strings.TrimPrefix(payload["location"], "#")
	if encoded == "" {
		encoded = strings.TrimPrefix(payload["data"], "#")
	}
	if encoded == "" {
		return OAuthTokenResult{}, fmt.Errorf("OpenList 回调缺少 token 数据")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		if raw, err = base64.RawStdEncoding.DecodeString(encoded); err != nil {
			return OAuthTokenResult{}, err
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return OAuthTokenResult{}, err
	}
	token := tokenResultFromMap(decoded)
	if !token.Done {
		return OAuthTokenResult{}, fmt.Errorf("OpenList 回调未返回访问凭证")
	}
	return token, nil
}

func (provider openListOAuthProvider) Poll(_ context.Context, _ string) (OAuthTokenResult, error) {
	return OAuthTokenResult{}, errUnsupportedOAuthOperation
}

type cloudDriveOAuthProvider struct{}

func (provider cloudDriveOAuthProvider) BuildAuth(_ context.Context, req OAuthURLRequest) (OAuthURLResult, error) {
	state := helpers.RandStr(16)
	SaveOAuthState(OAuthState{State: state, AccountID: req.AccountID, Provider: ProviderCloudDrive, RedirectURL: req.RedirectURL})
	values := url.Values{}
	values.Set("client_id", "100195313")
	values.Set("redirect_uri", "https://redirect115.zhenyunpan.com")
	values.Set("response_type", "code")
	values.Set("state", state)
	return OAuthURLResult{
		AuthURL:   "https://qrcodeapi.115.com/open/authorize?" + values.Encode(),
		State:     state,
		ExpiresIn: OAuthStateTTLSeconds,
	}, nil
}

func (provider cloudDriveOAuthProvider) Confirm(_ context.Context, payload map[string]string) (OAuthTokenResult, error) {
	state := payload["state"]
	if state != "" {
		if _, ok := GetOAuthState(state, ProviderCloudDrive); !ok {
			return OAuthTokenResult{}, fmt.Errorf("授权状态不存在或已过期")
		}
		defer DeleteOAuthState(state)
	}
	token := tokenResultFromPayload(payload)
	if !token.Done {
		return OAuthTokenResult{}, fmt.Errorf("CloudDrive 回调未返回访问凭证")
	}
	return token, nil
}

func (provider cloudDriveOAuthProvider) Poll(_ context.Context, _ string) (OAuthTokenResult, error) {
	return OAuthTokenResult{}, errUnsupportedOAuthOperation
}

func httpGetJSON(ctx context.Context, client *http.Client, endpoint string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("授权服务返回 HTTP %d: %s", resp.StatusCode, string(body))
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if nested, ok := data["data"].(map[string]any); ok {
		for key, value := range nested {
			if _, exists := data[key]; !exists {
				data[key] = value
			}
		}
	}
	return data, nil
}

func tokenResultFromPayload(payload map[string]string) OAuthTokenResult {
	expiresIn, _ := strconv.ParseInt(payload["expires_in"], 10, 64)
	token := OAuthTokenResult{
		AccessToken:  payload["access_token"],
		RefreshToken: payload["refresh_token"],
		ExpiresIn:    expiresIn,
	}
	token.Done = token.AccessToken != "" && token.RefreshToken != ""
	return token
}

func tokenResultFromMap(data map[string]any) OAuthTokenResult {
	if nested, ok := data["data"].(map[string]any); ok {
		for key, value := range nested {
			if _, exists := data[key]; !exists {
				data[key] = value
			}
		}
	}
	expiresIn := int64(0)
	switch value := data["expires_in"].(type) {
	case float64:
		expiresIn = int64(value)
	case json.Number:
		expiresIn, _ = value.Int64()
	case string:
		expiresIn, _ = strconv.ParseInt(value, 10, 64)
	}
	token := OAuthTokenResult{
		AccessToken:  stringField(data, "access_token"),
		RefreshToken: stringField(data, "refresh_token"),
		ExpiresIn:    expiresIn,
	}
	token.Done = token.AccessToken != "" && token.RefreshToken != ""
	return token
}

func stringField(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}
