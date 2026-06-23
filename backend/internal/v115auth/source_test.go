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
		ProviderOpenList:   "100197303",
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
