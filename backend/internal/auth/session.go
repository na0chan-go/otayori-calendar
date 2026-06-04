package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const SessionCookieName = "otayori_session"

type SessionManager struct {
	secret []byte
}

func NewSessionManager(secret string) SessionManager {
	return SessionManager{secret: []byte(secret)}
}

func (m SessionManager) Sign(userID uuid.UUID, now time.Time) string {
	expiresAt := now.Add(30 * 24 * time.Hour).Unix()
	payload := fmt.Sprintf("%s.%d", userID.String(), expiresAt)
	signature := m.signature(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "." + signature))
}

func (m SessionManager) Verify(value string, now time.Time) (uuid.UUID, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return uuid.Nil, err
	}

	parts := strings.Split(string(decoded), ".")
	if len(parts) != 3 {
		return uuid.Nil, errors.New("invalid session format")
	}

	payload := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(m.signature(payload))) {
		return uuid.Nil, errors.New("invalid session signature")
	}

	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return uuid.Nil, err
	}
	if now.After(time.Unix(expiresAt, 0)) {
		return uuid.Nil, errors.New("session expired")
	}

	return uuid.Parse(parts[0])
}

func (m SessionManager) signature(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
