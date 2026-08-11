package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

const encryptionKeyEnv = "WXX_ENCRYPTION_KEY"

var masterKey []byte

// ensureKey 懒加载加密密钥（首次调用时读取环境变量）。
// 未配置时返回错误，禁止敏感字段静默落为明文（P0-04）。
func ensureKey() ([]byte, error) {
	if masterKey != nil {
		return masterKey, nil
	}
	keyHex := os.Getenv(encryptionKeyEnv)
	if keyHex == "" {
		return nil, fmt.Errorf("环境变量 %s 未设置", encryptionKeyEnv)
	}
	h := sha256.Sum256([]byte(keyHex))
	masterKey = h[:]
	return masterKey, nil
}

// ValidateEncryptionKey 校验加密密钥是否已配置，未配置时返回错误。
// 由生产启动入口（cmd/server）调用，缺密钥即拒绝启动，避免敏感字段静默明文落库。
func ValidateEncryptionKey() error {
	if _, err := ensureKey(); err != nil {
		return fmt.Errorf("敏感字段加密不可用：%w", err)
	}
	return nil
}

func encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := ensureKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(append(nonce, ciphertext...)), nil
}

func decrypt(cipherHex string) (string, error) {
	if cipherHex == "" {
		return "", nil
	}
	key, err := ensureKey()
	if err != nil {
		return "", err
	}

	data, err := hex.DecodeString(cipherHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文数据过短")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
