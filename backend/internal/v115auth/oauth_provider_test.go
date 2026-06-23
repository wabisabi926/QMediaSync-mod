package v115auth

import (
	"Q115-STRM/internal/helpers"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuthProviderRegistry(t *testing.T) {
	for _, provider := range []AuthProvider{
		ProviderQMediaSync,
		ProviderMQFamily,
		ProviderMoviePilot,
		ProviderOpenList,
		ProviderCloudDrive,
	} {
		if _, ok := GetOAuthProvider(provider); !ok {
			t.Fatalf("未注册 OAuth provider: %s", provider)
		}
	}
}

func TestOAuthProviderRelayBuildAuth(t *testing.T) {
	oldKey := helpers.ENCRYPTION_KEY
	oldAuthServer := helpers.GlobalConfig.AuthServer
	oldNewAuthServer := helpers.GlobalConfig.NewAuthServer
	t.Cleanup(func() {
		helpers.ENCRYPTION_KEY = oldKey
		helpers.GlobalConfig.AuthServer = oldAuthServer
		helpers.GlobalConfig.NewAuthServer = oldNewAuthServer
	})

	helpers.ENCRYPTION_KEY = "shared-secret"
	helpers.GlobalConfig.NewAuthServer = "https://oauth.qmediasync.cn"
	provider, ok := GetOAuthProvider(ProviderQMediaSync)
	if !ok {
		t.Fatal("未找到 qmediasync provider")
	}
	result, err := provider.BuildAuth(context.Background(), OAuthURLRequest{
		AccountID:   1,
		AppID:       BuiltInRelayQMediaSync,
		RedirectURL: "http://127.0.0.1:1233",
		Provider:    ProviderQMediaSync,
	})
	if err != nil {
		t.Fatalf("生成内置中转授权地址失败: %v", err)
	}
	if !strings.HasPrefix(result.AuthURL, "https://oauth.qmediasync.cn/115.php?action=code&state=") {
		t.Fatalf("授权地址 = %s", result.AuthURL)
	}
	if result.Polling {
		t.Fatal("内置中转不应标记为轮询授权")
	}
}

func TestOAuthProviderRelayRequiresEncryptionKey(t *testing.T) {
	oldKey := helpers.ENCRYPTION_KEY
	t.Cleanup(func() { helpers.ENCRYPTION_KEY = oldKey })
	helpers.ENCRYPTION_KEY = ""

	provider, ok := GetOAuthProvider(ProviderMQFamily)
	if !ok {
		t.Fatal("未找到 mqfamily provider")
	}
	_, err := provider.BuildAuth(context.Background(), OAuthURLRequest{
		AccountID:   1,
		AppID:       BuiltInRelayQ115STRM,
		RedirectURL: "http://127.0.0.1:1233",
		Provider:    ProviderMQFamily,
	})
	if err == nil {
		t.Fatal("缺少 ENCRYPTION_KEY 时应返回错误")
	}
}

func TestOAuthProviderOpenListBuildAuthParsesTextURL(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"https://115.com/?ac=open&redirect_uri=https%3A%2F%2Fpassportapi.115.com%2Fopen%2Fauthorize%3Fclient_id%3D100197303%26state%3Dopenlist-state%26response_type%3Dcode%26redirect_uri%3Dhttps%253A%252F%252Fapi.oplist.org%252F115cloud%252Fcallback"}`))
	}))
	defer server.Close()

	provider := openListOAuthProvider{client: server.Client(), authServer: server.URL}
	result, err := provider.BuildAuth(context.Background(), OAuthURLRequest{
		AccountID:   3,
		RedirectURL: "http://127.0.0.1:12333/#/cloud-accounts",
		Provider:    ProviderOpenList,
	})
	if err != nil {
		t.Fatalf("生成 OpenList 授权地址失败: %v", err)
	}
	if requestedPath != "/115cloud/requests?driver_txt=115cloud_go&server_use=true" {
		t.Fatalf("OpenList 请求路径 = %s", requestedPath)
	}
	if !strings.HasPrefix(result.AuthURL, "https://115.com/?ac=open&redirect_uri=") {
		t.Fatalf("OpenList 授权地址 = %s", result.AuthURL)
	}
	outer, err := url.Parse(result.AuthURL)
	if err != nil {
		t.Fatalf("解析 OpenList 授权地址失败: %v", err)
	}
	inner, err := url.Parse(outer.Query().Get("redirect_uri"))
	if err != nil {
		t.Fatalf("解析 OpenList 内层授权地址失败: %v", err)
	}
	if result.State != "openlist-state" || inner.Query().Get("state") != "openlist-state" {
		t.Fatalf("OpenList state = %q，内层 state = %q", result.State, inner.Query().Get("state"))
	}
	if _, ok := GetOAuthState("openlist-state", ProviderOpenList); !ok {
		t.Fatal("OpenList state 未保存")
	}
	DeleteOAuthState("openlist-state")
}

func TestOAuthProviderCloudDriveBuildAuthUnavailable(t *testing.T) {
	provider := cloudDriveOAuthProvider{}
	_, err := provider.BuildAuth(context.Background(), OAuthURLRequest{
		AccountID: 4,
		Provider:  ProviderCloudDrive,
	})
	if err == nil || !strings.Contains(err.Error(), "CloudDrive 网页授权服务当前不可用") {
		t.Fatalf("CloudDrive 不可用错误 = %v", err)
	}
}
