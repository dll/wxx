package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

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

// TestCryptoRoundTrip 校验 encrypt/decrypt 对称性（有密钥时密文≠明文）
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
	// 无密钥降级时明文存储；有密钥时密文 ≠ 明文
	if masterKey != nil && enc == plain {
		t.Fatal("有密钥时密文不应等于明文")
	}
}
