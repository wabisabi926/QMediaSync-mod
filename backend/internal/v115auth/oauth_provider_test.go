package v115auth

import (
	"Q115-STRM/internal/helpers"
	"context"
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
