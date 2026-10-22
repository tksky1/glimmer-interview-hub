package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 旧系统下发的 JWT 载荷（线上版本含 role 字段）
type Claims struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	StudentID string `json:"studentid"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// Parse 校验并解析 JWT（HS256，密钥可配）
func Parse(tokenStr, secret string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Username == "" {
		return nil, errors.New("token missing username")
	}
	return claims, nil
}

// DecodeUnverified 不验签解析载荷（用于委派校验前提取身份）。
// 注意：调用方必须已通过旧系统 API 确认 token 有效，才能信任其结果。
func DecodeUnverified(tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	claims := &Claims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenStr, claims)
	if err != nil {
		return nil, err
	}
	if claims.Username == "" {
		return nil, errors.New("token missing username")
	}
	return claims, nil
}

// IsExpired 检查载荷是否已过期（无 exp 声明时视为不过期）
func (c *Claims) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(c.ExpiresAt.Time)
}
