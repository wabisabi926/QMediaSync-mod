package helpers

import (
	"os"
	"path/filepath"
	"testing"
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

func TestEnsureJWTSecretReplacesEmptyAndDefault(t *testing.T) {
	cases := []struct {
		name      string
		secret    string
		wantRenew bool
	}{
		{name: "空值", secret: "", wantRenew: true},
		{name: "公开默认值", secret: DefaultJWTSecret, wantRenew: true},
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
