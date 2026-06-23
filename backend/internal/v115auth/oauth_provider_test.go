package v115auth

import (
	"Q115-STRM/internal/helpers"
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestOAuthProviderRegistry(t *testing.T) {
	for _, provider := range []AuthProvider{
		ProviderQMediaSync,
		ProviderMQFamily,
		ProviderMoviePilot,
		ProviderCloudDrive,
	} {
		if _, ok := GetOAuthProvider(provider); !ok {
			t.Fatalf("未注册 OAuth provider: %s", provider)
		}
	}
	if _, ok := GetOAuthProvider(ProviderOpenList); ok {
		t.Fatal("OpenList 网页授权已删除，不应注册 OAuth provider")
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

func TestOAuthProviderCloudDriveBuildAuthUsesRedirectState(t *testing.T) {
	provider := cloudDriveOAuthProvider{}
	result, err := provider.BuildAuth(context.Background(), OAuthURLRequest{
		AccountID:   4,
		RedirectURL: "http://127.0.0.1:12333/#/cloud-accounts",
		Provider:    ProviderCloudDrive,
	})
	if err != nil {
		t.Fatalf("生成 CloudDrive 授权地址失败: %v", err)
	}
	authURL, err := url.Parse(result.AuthURL)
	if err != nil {
		t.Fatalf("解析 CloudDrive 授权地址失败: %v", err)
	}
	query := authURL.Query()
	if authURL.Scheme != "https" || authURL.Host != "passportapi.115.com" || authURL.Path != "/open/authorize" {
		t.Fatalf("CloudDrive 授权地址 = %s", result.AuthURL)
	}
	if query.Get("client_id") != "100195313" {
		t.Fatalf("client_id = %s", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "https://redirect115.zhenyunpan.com" {
		t.Fatalf("redirect_uri = %s", query.Get("redirect_uri"))
	}
	if query.Get("response_type") != "code" {
		t.Fatalf("response_type = %s", query.Get("response_type"))
	}
	if query.Get("state") != "http://127.0.0.1:12333/#/cloud-accounts?account_id=4&source=115" {
		t.Fatalf("state = %s", query.Get("state"))
	}
}
