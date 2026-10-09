package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type CriptoTokenManager struct {
	Key []byte
}

type TokenData struct {
	SessionID string
	UserID    int
	Exp       int64
}

func NewCriptoTokenManager(key []byte) *CriptoTokenManager {
	return &CriptoTokenManager{
		Key: key,
	}
}

func (tm *CriptoTokenManager) Create(sess *Session, expUnix int64) (string, error) {
	block, _ := aes.NewCipher(tm.Key)

	gcm, _ := cipher.NewGCM(block)

	td := TokenData{SessionID: sess.ID, UserID: sess.UserID, Exp: expUnix}
	plain, _ := json.Marshal(td)

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	chiperText := gcm.Seal(nil, nonce, plain, nil)

	raw := append(nonce, chiperText...)

	token := base64.StdEncoding.EncodeToString(raw)

	return token, nil

}

func (tm *CriptoTokenManager) Check(sess *Session, token string) (bool, error) {

	block, _ := aes.NewCipher(tm.Key)
	gcm, _ := cipher.NewGCM(block)
	nonceLen := gcm.NonceSize()

	parts, _ := base64.StdEncoding.DecodeString(token)
	nonce := parts[:nonceLen]
	ciphertext := parts[nonceLen:]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return false, err
	}

	var td TokenData
	json.Unmarshal(plain, &td)

	if td.Exp < time.Now().Unix() {
		return false, fmt.Errorf("Token expired")
	}

	ok := td.SessionID == sess.ID && td.UserID == sess.UserID
	if !ok {
		return false, fmt.Errorf("token invalid")
	}

	return true, nil

}
