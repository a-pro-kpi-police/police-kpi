package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const tokenTTL = 12 * time.Hour

type tokenClaims struct {
	UserID int
	Role   string
	Exp    int64
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2Key([]byte(password), salt, 100_000, 32, sha256.New)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(key), nil
}

func verifyPassword(password, stored string) bool {
	parts := strings.Split(stored, ":")
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	actual := pbkdf2Key([]byte(password), salt, 100_000, len(expected), sha256.New)
	return hmac.Equal(actual, expected)
}

func pbkdf2Key(password, salt []byte, iter, keyLen int, h func() hash.Hash) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var output []byte
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		_, _ = prf.Write(salt)
		_, _ = prf.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := prf.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iter; i++ {
			prf.Reset()
			_, _ = prf.Write(u)
			u = prf.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		output = append(output, t...)
	}
	return output[:keyLen]
}

func (a *App) createToken(user *User) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]any{
		"sub":  strconv.Itoa(user.ID),
		"role": user.Role,
		"exp":  time.Now().UTC().Add(tokenTTL).Unix(),
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerBytes) + "." +
		base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := signHS256([]byte(unsigned), a.secret)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (a *App) parseToken(token string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token")
	}
	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	expected := signHS256([]byte(unsigned), a.secret)
	if !hmac.Equal(signature, expected) {
		return nil, errors.New("invalid signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		return nil, err
	}
	sub, _ := raw["sub"].(string)
	userID, err := strconv.Atoi(sub)
	if err != nil {
		return nil, err
	}
	role, _ := raw["role"].(string)
	exp, ok := raw["exp"].(float64)
	if !ok || int64(exp) < time.Now().UTC().Unix() {
		return nil, errors.New("expired token")
	}
	return &tokenClaims{UserID: userID, Role: role, Exp: int64(exp)}, nil
}

func signHS256(message []byte, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(message)
	return mac.Sum(nil)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректная форма входа")
		return
	}
	username := r.Form.Get("username")
	password := r.Form.Get("password")
	user, err := a.getUserByUsername(context.Background(), username)
	if err != nil || !verifyPassword(password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}
	token, err := a.createToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Не удалось создать токен: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, TokenOut{AccessToken: token, TokenType: "bearer"})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, userOut(user))
}
