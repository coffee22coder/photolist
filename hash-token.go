package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type HashTokenManager struct {
	Secret []byte
}

func NewHashTokenManager(s []byte) *HashTokenManager {
	return &HashTokenManager{
		Secret: s,
	}
}

func (tm *HashTokenManager) Create(sess *Session, expUnix int64) (string, error) {
	payload := fmt.Sprintf("%d:%s:%d", sess.UserID, sess.ID, expUnix)

	hash := hmac.New(sha256.New, tm.Secret)
	hash.Write([]byte(payload))
	signature := hex.EncodeToString(hash.Sum(nil))

	return fmt.Sprintf("%s:%d", signature, expUnix), nil
}

func (tm *HashTokenManager) Check(sess *Session, token string) (bool, error) {
	tokenParts := strings.Split(token, ":")
	if len(tokenParts) != 2 || tokenParts[0] == "" || tokenParts[1] == "" {
		return false, fmt.Errorf("token invalid")
	}
	nowUnix := time.Now().Unix()
	expUnix, err := strconv.ParseInt(tokenParts[1], 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid expiration time: %w", err)
	}

	if expUnix < nowUnix {
		return false, nil
	}

	payload := fmt.Sprintf("%d:%s:%d", sess.UserID, sess.ID, expUnix)
	hash := hmac.New(sha256.New, tm.Secret)
	hash.Write([]byte(payload))
	expected := hex.EncodeToString(hash.Sum(nil))

	tokenMac, err := hex.DecodeString(tokenParts[0])
	if err != nil {
		return false, fmt.Errorf("invalid hex in token: %w", err)
	}
	expectedMac, err := hex.DecodeString(expected)
	if err != nil {
		return false, fmt.Errorf("invalid hex in token: %w", err)
	}

	if hmac.Equal(tokenMac, expectedMac) {
		return true, nil
	}

	return false, fmt.Errorf("token invalid")
}
