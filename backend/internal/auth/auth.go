package auth

import (
	"log"
	"time"

	"google-ai-proxy/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SecretKey JWT 签名密钥，通过 InitSecretKey() 从环境变量加载
var SecretKey []byte

// LicenseSecretKey License Key 签名密钥，通过 InitSecretKey() 从环境变量加载
var LicenseSecretKey []byte

// InitSecretKey 从环境变量 JWT_SECRET 初始化密钥，必须在加载 .env 之后调用
func InitSecretKey() {
	secret := config.GetJWTSecret()
	if secret == "" {
		log.Fatal("JWT_SECRET 环境变量未设置，服务无法启动")
	}
	SecretKey = []byte(secret)
	LicenseSecretKey = []byte(secret)
	log.Println("JWT 密钥已加载")
}

// LicenseClaims 用于解析License密钥
type LicenseClaims struct {
	ID      string `json:"id"`
	Credits int    `json:"credits"`
	jwt.RegisteredClaims
}

// UserClaims 用于用户JWT认证
type UserClaims struct {
	UserID uint64 `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateLicenseKey(credits int) (string, error) {
	id := uuid.New().String()
	claims := &LicenseClaims{
		ID:               id,
		Credits:          credits,
		RegisteredClaims: jwt.RegisteredClaims{
			// No expiration for the key itself, or maybe long expiration?
			// Let's say valid for 10 years for now.
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(LicenseSecretKey)
}

func ValidateLicenseKey(tokenString string) (*LicenseClaims, error) {
	claims := &LicenseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return LicenseSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, err
	}

	return claims, nil
}

// GenerateUserToken 生成用户登录令牌 (7天有效期)
func GenerateUserToken(userID uint64, email string) (string, error) {
	claims := &UserClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

// ValidateUserToken 验证用户令牌
func ValidateUserToken(tokenString string) (*UserClaims, error) {
	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenExpired
	}

	return claims, nil
}

// HashPassword 密码哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
