package repository

import (
	"os"
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

// TestMain 注入加密密钥：crypto.go 已删除明文降级（P0-04），
// 无密钥时 encrypt/decrypt 直接返回错误，故测试必须显式提供密钥。
func TestMain(m *testing.M) {
	os.Setenv(encryptionKeyEnv, "test-encryption-key-32bytes")
	code := m.Run()
	os.Unsetenv(encryptionKeyEnv)
	os.Exit(code)
}

// TestPortalCredentialRepo_Store 校验门户凭证仓库功能正确（存/取/覆盖/删）
func TestPortalCredentialRepo_Store(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	r := NewPortalCredentialRepo(db)

	if err := r.Upsert(1, "https://my0.chzu.edu.cn/", "20240001", "MySecret@123"); err != nil {
		t.Fatalf("Upsert 失败: %v", err)
	}
	c, err := r.Get(1)
	if err != nil || c == nil {
		t.Fatalf("Get 失败: %v c=%v", err, c)
	}
	if c.PortalPasswordEnc == "" {
		t.Fatal("加密字段不应为空")
	}
	if c.PortalAccount != "20240001" {
		t.Fatalf("账号错误: %s", c.PortalAccount)
	}

	// 覆盖更新 → 密文应变化（AES-GCM 随机 nonce）
	if err := r.Upsert(1, "https://my0.chzu.edu.cn/", "20240001", "NewPass456"); err != nil {
		t.Fatalf("Upsert2 失败: %v", err)
	}
	c2, _ := r.Get(1)
	if c2.PortalPasswordEnc == c.PortalPasswordEnc {
		t.Fatal("覆盖后密文应变化（nonce 随机）")
	}

	if err := r.Delete(1); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	c3, _ := r.Get(1)
	if c3 != nil {
		t.Fatal("删除后应无记录")
	}
}

// TestCryptoRoundTrip 校验 encrypt/decrypt 对称性（配置密钥后密文 ≠ 明文）
func TestCryptoRoundTrip(t *testing.T) {
	plain := "SuperSecret#2026"
	enc, err := encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt 失败: %v", err)
	}
	dec, err := decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt 失败: %v", err)
	}
	if dec != plain {
		t.Fatalf("解密结果不一致: %s != %s", dec, plain)
	}
	// P0-04：有密钥时必须真正加密，密文不得等于明文
	if enc == plain {
		t.Fatal("有密钥时密文不应等于明文")
	}
}
