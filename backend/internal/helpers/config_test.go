package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestInitConfigReadsConfigYaml(t *testing.T) {
	withTempConfigDir(t, func(configDir string) {
		writeTestConfig(t, filepath.Join(configDir, "config.yaml"), "from-yaml")

		if err := InitConfig(); err != nil {
			t.Fatalf("InitConfig() error = %v", err)
		}

		if GlobalConfig.JwtSecret != "from-yaml" {
			t.Fatalf("GlobalConfig.JwtSecret = %q, want %q", GlobalConfig.JwtSecret, "from-yaml")
		}
	})
}

func TestInitConfigFallsBackToConfigYml(t *testing.T) {
	withTempConfigDir(t, func(configDir string) {
		writeTestConfig(t, filepath.Join(configDir, "config.yml"), "from-yml")

		if err := InitConfig(); err != nil {
			t.Fatalf("InitConfig() error = %v", err)
		}

		if GlobalConfig.JwtSecret != "from-yml" {
			t.Fatalf("GlobalConfig.JwtSecret = %q, want %q", GlobalConfig.JwtSecret, "from-yml")
		}
	})
}

func TestInitConfigPrefersConfigYamlOverConfigYml(t *testing.T) {
	withTempConfigDir(t, func(configDir string) {
		writeTestConfig(t, filepath.Join(configDir, "config.yaml"), "from-yaml")
		writeTestConfig(t, filepath.Join(configDir, "config.yml"), "from-yml")

		if err := InitConfig(); err != nil {
			t.Fatalf("InitConfig() error = %v", err)
		}

		if GlobalConfig.JwtSecret != "from-yaml" {
			t.Fatalf("GlobalConfig.JwtSecret = %q, want %q", GlobalConfig.JwtSecret, "from-yaml")
		}
	})
}

func TestExistingConfigFilePathDefaultsToConfigYaml(t *testing.T) {
	withTempConfigDir(t, func(configDir string) {
		got := ExistingConfigFilePath()
		want := filepath.Join(configDir, "config.yaml")

		if got != want {
			t.Fatalf("ExistingConfigFilePath() = %q, want %q", got, want)
		}
	})
}

func TestSaveConfigWritesConfigYaml(t *testing.T) {
	withTempConfigDir(t, func(configDir string) {
		cfg := MakeDefaultConfig()
		cfg.JwtSecret = "saved-yaml"

		if err := SaveConfig(cfg); err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		if !PathExists(filepath.Join(configDir, "config.yaml")) {
			t.Fatal("SaveConfig() did not create config.yaml")
		}
		if PathExists(filepath.Join(configDir, "config.yml")) {
			t.Fatal("SaveConfig() created legacy config.yml")
		}
	})
}

func TestMakeDefaultConfigDisablesEmby302InsecureSkipVerify(t *testing.T) {
	cfg := MakeDefaultConfig()
	if cfg.Emby302.InsecureSkipVerify {
		t.Fatal("Emby302.InsecureSkipVerify = true, want false")
	}
}

func TestMakeDefaultConfigDoesNotContainAdminCredentials(t *testing.T) {
	cfg := MakeDefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal default config: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "adminUsername") || strings.Contains(text, "adminPassword") {
		t.Fatalf("默认配置不应包含管理员字段: %s", text)
	}
}

func TestInitConfigReadsEmby302InsecureSkipVerify(t *testing.T) {
	withTempConfigDir(t, func(configDir string) {
		data := []byte("jwtSecret: custom-secret\nemby302:\n  insecure_skip_verify: true\n")
		if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0644); err != nil {
			t.Fatalf("write test config: %v", err)
		}

		if err := InitConfig(); err != nil {
			t.Fatalf("InitConfig() error = %v", err)
		}
		if !GlobalConfig.Emby302.InsecureSkipVerify {
			t.Fatal("Emby302.InsecureSkipVerify = false, want true")
		}
	})
}

func TestEnsureJWTSecretReplacesEmptyAndDefault(t *testing.T) {
	cases := []struct {
		name      string
		secret    string
		wantRenew bool
	}{
		{name: "空值", secret: "", wantRenew: true},
		{name: "公开默认值", secret: DefaultJWTSecret, wantRenew: true},
		{name: "旧公开默认值", secret: "Q115-STRM-JWT-TOKEN-250706", wantRenew: true},
		{name: "自定义值", secret: "custom-secret-value", wantRenew: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTempConfigDir(t, func(configDir string) {
				GlobalConfig = *MakeDefaultConfig()
				GlobalConfig.JwtSecret = tc.secret

				if err := EnsureJWTSecret(); err != nil {
					t.Fatalf("EnsureJWTSecret() error = %v", err)
				}
				if GlobalConfig.JwtSecret == "" || GlobalConfig.JwtSecret == DefaultJWTSecret {
					t.Fatalf("JwtSecret 未替换为强随机值: %q", GlobalConfig.JwtSecret)
				}
				if !tc.wantRenew && GlobalConfig.JwtSecret != tc.secret {
					t.Fatalf("自定义 JwtSecret 被错误替换: %q", GlobalConfig.JwtSecret)
				}
				if tc.wantRenew && len(GlobalConfig.JwtSecret) < 43 {
					t.Fatalf("JwtSecret 长度过短: %d", len(GlobalConfig.JwtSecret))
				}
			})
		})
	}
}

func withTempConfigDir(t *testing.T, run func(configDir string)) {
	t.Helper()

	oldConfigDir := ConfigDir
	oldGlobalConfig := GlobalConfig
	t.Cleanup(func() {
		ConfigDir = oldConfigDir
		GlobalConfig = oldGlobalConfig
	})

	ConfigDir = t.TempDir()
	GlobalConfig = Config{}
	run(ConfigDir)
}

func writeTestConfig(t *testing.T, path, jwtSecret string) {
	t.Helper()

	data := []byte("jwtSecret: " + jwtSecret + "\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write test config %s: %v", path, err)
	}
}
