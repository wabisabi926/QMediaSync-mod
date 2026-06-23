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
