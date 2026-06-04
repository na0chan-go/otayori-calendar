package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type TokenCipher struct {
	gcm cipher.AEAD
}

func NewTokenCipher(secret string) (TokenCipher, error) {
	if secret == "" {
		return TokenCipher{}, errors.New("token cipher secret is required")
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return TokenCipher{}, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return TokenCipher{}, err
	}

	return TokenCipher{gcm: gcm}, nil
}

func (c TokenCipher) Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := c.gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.RawURLEncoding.EncodeToString(cipherText), nil
}

func (c TokenCipher) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	cipherText, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(cipherText) < c.gcm.NonceSize() {
		return "", errors.New("cipher text is too short")
	}

	nonce := cipherText[:c.gcm.NonceSize()]
	encrypted := cipherText[c.gcm.NonceSize():]
	plainText, err := c.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
