package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const encryptedOAuthTokenPrefix = "enc:v1:"

var ErrOAuthTokenEncryptionKeyMissing = errors.New("oauth token encryption key is missing")

type oauthTokenCipher struct {
	key []byte
}

func newOAuthTokenCipher(secret string) *oauthTokenCipher {
	if secret == "" {
		return &oauthTokenCipher{}
	}
	sum := sha256.Sum256([]byte(secret))
	return &oauthTokenCipher{key: sum[:]}
}

func (c *oauthTokenCipher) enabled() bool {
	return c != nil && len(c.key) > 0
}

func (c *oauthTokenCipher) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if !c.enabled() {
		return "", ErrOAuthTokenEncryptionKeyMissing
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedOAuthTokenPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *oauthTokenCipher) decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, encryptedOAuthTokenPrefix) {
		return stored, nil
	}
	if !c.enabled() {
		return "", ErrOAuthTokenEncryptionKeyMissing
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(stored, encryptedOAuthTokenPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("encrypted oauth token is too short")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
