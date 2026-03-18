package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 存放我们要签发到身份证里的信息
type CustomClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 登录成功时调用，颁发身份证
func GenerateToken(username string, secret string, expireDuration time.Duration) (string, error) {
	claims := CustomClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "quiz-server",
		},
	}
	// 使用 HS256 算法和你的专属密钥进行签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 每次用户请求时调用，验证身份证真伪
func ParseToken(tokenString string, secret string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("无效的 token")
}