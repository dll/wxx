package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"

	"crypto/sha256"
)

const encryptionKeyEnv = "WXX_ENCRYPTION_KEY"

var masterKey []byte

func init() {
	keyHex := os.Getenv(encryptionKeyEnv)
	if keyHex == "" {
		log.Printf("[WARN] WXX_ENCRYPTION_KEY 未设置，密钥将以明文存储")
		return
	}
	h := sha256.Sum256([]byte(keyHex))
	masterKey = h[:]
}

func encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if masterKey == nil {
		return plaintext, nil
	}

	block, err := aes.NewCipher(masterKey)
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
	if masterKey == nil {
		return cipherHex, nil
	}

	data, err := hex.DecodeString(cipherHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(masterKey)
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
