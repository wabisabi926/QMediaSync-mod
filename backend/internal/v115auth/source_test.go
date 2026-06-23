package v115auth

import "testing"

func TestBuiltInAppIDOrder(t *testing.T) {
	sources := BuiltInAppIDSources()
	want := []string{"QMediaSync", "Q115-STRM", "MQ的媒体库"}
	for i, name := range want {
		if sources[i].AppName != name {
			t.Fatalf("第 %d 个内置 APPID = %s，期望 %s", i, sources[i].AppName, name)
		}
		if sources[i].SourceType != SourceTypeBuiltInAppID {
			t.Fatalf("来源类型 = %s，期望 %s", sources[i].SourceType, SourceTypeBuiltInAppID)
		}
		if sources[i].Provider != ProviderOfficialPKCE {
			t.Fatalf("授权 provider = %s，期望 %s", sources[i].Provider, ProviderOfficialPKCE)
		}
		if sources[i].RequiresEncryptionKey {
			t.Fatal("内置 APPID 不应要求共享 ENCRYPTION_KEY")
		}
	}
}

func TestBuiltInAppIDCatalogCompleteness(t *testing.T) {
	sources := BuiltInAppIDSources()
	if len(sources) != 1128 {
		t.Fatalf("内置 APPID 目录数量 = %d，期望 1128", len(sources))
	}

	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if _, ok := seen[source.AppID]; ok {
			t.Fatalf("内置 APPID 目录存在重复项: %s", source.AppID)
		}
		seen[source.AppID] = struct{}{}
	}

	if _, ok := seen["100197925"]; !ok {
		t.Fatal("内置 APPID 目录缺少 p115client OPEN_APP_IDS 尾部项 100197925")
	}
}

func TestSearchBuiltInAppIDSourcesFindsTailAppID(t *testing.T) {
	result := SearchBuiltInAppIDSources("100197925", 0, 50)
	if result.Total != 1 {
		t.Fatalf("搜索尾部 APPID 总数 = %d，期望 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("搜索尾部 APPID 返回数量 = %d，期望 1", len(result.Items))
	}
	if result.Items[0].AppID != "100197925" {
		t.Fatalf("搜索尾部 APPID 返回 %s，期望 100197925", result.Items[0].AppID)
	}
	if result.Items[0].AppName != "自己文件处理2" {
		t.Fatalf("搜索尾部 APPID 名称 = %s，期望 自己文件处理2", result.Items[0].AppName)
	}
}

func TestKnownThirdPartySources(t *testing.T) {
	cases := map[AuthProvider]string{
		ProviderMoviePilot: "100197847",
		ProviderCloudDrive: "100195313",
	}
	for provider, appID := range cases {
		source, ok := FindSource(SourceTypeThirdPartyService, provider, appID)
		if !ok {
			t.Fatalf("未找到第三方来源 %s", provider)
		}
		if source.AppID != appID {
			t.Fatalf("%s APPID = %s，期望 %s", provider, source.AppID, appID)
		}
		if source.RequiresEncryptionKey {
			t.Fatalf("%s 不应要求共享 ENCRYPTION_KEY", provider)
		}
	}
}

func TestResolveLegacyRelayAccount(t *testing.T) {
	cases := []struct {
		appID    string
		provider AuthProvider
	}{
		{"QMediaSync", ProviderQMediaSync},
		{"Q115-STRM", ProviderMQFamily},
		{"MQ的媒体库", ProviderMQFamily},
	}
	for _, tt := range cases {
		source := ResolveAccountSource(tt.appID, "")
		if source.SourceType != SourceTypeBuiltInRelay {
			t.Fatalf("%s 来源类型 = %s，期望 %s", tt.appID, source.SourceType, SourceTypeBuiltInRelay)
		}
		if source.Provider != tt.provider {
			t.Fatalf("%s provider = %s，期望 %s", tt.appID, source.Provider, tt.provider)
		}
	}
}

func TestResolveNumericAppIDAccount(t *testing.T) {
	source := ResolveAccountSource("100197849", "QMediaSync")
	if source.SourceType != SourceTypeBuiltInAppID {
		t.Fatalf("来源类型 = %s，期望 %s", source.SourceType, SourceTypeBuiltInAppID)
	}
	if source.Provider != ProviderOfficialPKCE {
		t.Fatalf("provider = %s，期望 %s", source.Provider, ProviderOfficialPKCE)
	}
}
