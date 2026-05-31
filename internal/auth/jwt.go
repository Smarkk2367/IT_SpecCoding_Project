package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Subject  string  `json:"sub"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	ClientID *string `json:"client_id,omitempty"`
	IssuedAt int64   `json:"iat"`
	Expires  int64   `json:"exp"`
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) TokenManager {
	return TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m TokenManager) Issue(claims Claims) (string, error) {
	now := time.Now().UTC()
	claims.IssuedAt = now.Unix()
	claims.Expires = now.Add(m.ttl).Unix()

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := encodedHeader + "." + encodedClaims

	return signingInput + "." + m.sign(signingInput), nil
}

func (m TokenManager) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(m.sign(signingInput))) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Subject == "" || claims.Email == "" || claims.Role == "" {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().UTC().Unix() >= claims.Expires {
		return Claims{}, fmt.Errorf("%w: expired", ErrInvalidToken)
	}

	return claims, nil
}

func (m TokenManager) sign(input string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
