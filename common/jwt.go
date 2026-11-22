package common

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/coder9527-lg/go-user-service/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// 令牌类型，写在 Claims 中，防止 refresh token 被当作 access token 使用。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// 两种令牌的有效期：access 短时效降低泄露风险，refresh 长时效减少重新登录。
const (
	accessTokenTTL  = 2 * time.Hour
	refreshTokenTTL = 7 * 24 * time.Hour
)

// Claims 自定义 JWT 载荷。
type Claims struct {
	UserID    uint   `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair 登录成功后下发的一对令牌。
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token 有效期（秒）
}

var (
	randomSecretOnce sync.Once
	randomSecret     []byte
)

// jwtSecret 从配置读取签名密钥；未配置时生成一次性随机密钥。
// 随机密钥仅适合本地开发：进程重启后历史 token 全部失效。
// 生产环境必须通过配置项 jwt.secret 或环境变量 JWT_SECRET 注入稳定密钥。
func jwtSecret() []byte {
	if secret := viper.GetString("jwt.secret"); secret != "" {
		return []byte(secret)
	}
	randomSecretOnce.Do(func() {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			// rand.Read 在主流系统上不会失败；若真失败，直接 panic，避免用可预测密钥。
			panic(err)
		}
		randomSecret = []byte(hex.EncodeToString(buf))
		logrus.Warn("jwt.secret is empty: generated an ephemeral random secret; set JWT_SECRET in production")
	})
	return randomSecret
}

// sign 用指定类型和有效期签发一个 JWT。
func sign(userID uint, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	// 每次签发生成唯一 jti：保证同一秒内的令牌也不相同，
	// 同时为未来的令牌吊销/黑名单能力预留标识。
	jtiBuf := make([]byte, 16)
	if _, err := rand.Read(jtiBuf); err != nil {
		return "", err
	}
	claims := &Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        hex.EncodeToString(jtiBuf),
			Issuer:    "go-user-service",
			Subject:   tokenType + " token",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// IssueTokenPair 为用户签发一对 access + refresh 令牌。
func IssueTokenPair(user model.User) (*TokenPair, error) {
	access, err := sign(user.ID, TokenTypeAccess, accessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := sign(user.ID, TokenTypeRefresh, refreshTokenTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

// ReleaseToken 兼容旧调用：仅签发 access token。
func ReleaseToken(user model.User) (string, error) {
	return sign(user.ID, TokenTypeAccess, accessTokenTTL)
}

// RefreshAccessToken 用合法的 refresh token 换新的 access token；
// 拒绝把 access token 或其他类型令牌用于刷新。
func RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := ParseToken(refreshToken)
	if err != nil {
		return "", err
	}
	if claims.TokenType != TokenTypeRefresh {
		return "", errors.New("refresh token required")
	}
	return sign(claims.UserID, TokenTypeAccess, accessTokenTTL)
}

// ParseToken 解析并校验 JWT，返回其中的用户载荷。
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
