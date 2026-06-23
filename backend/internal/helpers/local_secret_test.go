package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempLocalEncryptionKey(t *testing.T, run func()) {
	t.Helper()
	oldConfigDir := ConfigDir
	oldLocalKey := localEncryptionKey
	oldRelayKey := OAuthRelayEncryptionKey
	t.Cleanup(func() {
		ConfigDir = oldConfigDir
		localEncryptionKey = oldLocalKey
		OAuthRelayEncryptionKey = oldRelayKey
	})
	ConfigDir = t.TempDir()
	localEncryptionKey = ""
	OAuthRelayEncryptionKey = ""
	run()
}

func TestInitEncryptionKeyCreatesAndReusesLocalKey(t *testing.T) {
	withTempLocalEncryptionKey(t, func() {
		if err := InitEncryptionKey(); err != nil {
			t.Fatalf("初始化本机加密密钥失败: %v", err)
		}
		firstKey := localEncryptionKey
		if firstKey == "" {
			t.Fatal("本机加密密钥不应为空")
		}
		keyPath := filepath.Join(ConfigDir, "encryption.key")
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatalf("读取本机加密密钥文件失败: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("本机加密密钥文件权限 = %v，期望 0600", info.Mode().Perm())
		}

		localEncryptionKey = ""
		if err := InitEncryptionKey(); err != nil {
			t.Fatalf("重新加载本机加密密钥失败: %v", err)
		}
		if localEncryptionKey != firstKey {
			t.Fatal("重新加载后应复用已有本机加密密钥")
		}
	})
}

func TestEncryptLocalSecretDoesNotRequireSharedEncryptionKey(t *testing.T) {
	withTempLocalEncryptionKey(t, func() {
		if err := InitEncryptionKey(); err != nil {
			t.Fatalf("初始化本机加密密钥失败: %v", err)
		}

		encrypted, err := EncryptLocalSecret("totp-secret")
		if err != nil {
			t.Fatalf("加密本机 secret 不应依赖 115 中转共享密钥: %v", err)
		}
		if !strings.HasPrefix(encrypted, "gcm:") {
			t.Fatalf("本机 secret 应使用 GCM 格式，实际为 %s", encrypted)
		}
		decrypted, err := DecryptLocalSecret(encrypted)
		if err != nil {
			t.Fatalf("解密本机 secret 失败: %v", err)
		}
		if decrypted != "totp-secret" {
			t.Fatalf("解密结果 = %s，期望 totp-secret", decrypted)
		}
	})
}

func TestDecryptLocalSecretRejectsRelayCBCData(t *testing.T) {
	withTempLocalEncryptionKey(t, func() {
		if err := InitEncryptionKey(); err != nil {
			t.Fatalf("初始化本机加密密钥失败: %v", err)
		}
		encrypted, err := EncryptWithKey("legacy-totp-secret", "relay-secret")
		if err != nil {
			t.Fatalf("准备中转 CBC 数据失败: %v", err)
		}

		if _, err := DecryptLocalSecret(encrypted); err == nil {
			t.Fatal("本机 secret 不应兼容 115 中转 CBC 数据")
		}
	})
}
