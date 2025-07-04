package auth

import (
	"errors"
	"log"
	"os"
	"time"

	"portal/repository/auth"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthService struct {
	authRepo *auth.AuthRepository
}

type LoginResult struct {
	Token   string `json:"token"`
	UserID  string `json:"user_id"`
	IsAdmin uint8  `json:"is_admin"`
	Email   string `json:"email"` // 新增Email字段
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		authRepo: auth.NewAuthRepository(db),
	}
}

type Claims struct {
	UserID  string `json:"user_id"`
	IsAdmin uint8  `json:"is_admin"`
	jwt.RegisteredClaims
}

// 移除 init 函数 - 不再尝试加载环境变量

// Login 处理用户登录认证
func (s *AuthService) Login(email, password string) (*LoginResult, error) {
	// 验证用户凭证
	user, err := s.authRepo.ValidateCredentials(email, password)
	if err != nil {
		return nil, err
	}

	// 从环境变量获取 JWT 配置
	jwtSecret := os.Getenv("JWT_SECRET")
	jwtExpireStr := os.Getenv("JWT_EXPIRE")

	// 如果环境变量为空，记录警告并使用默认值
	if jwtSecret == "" {
		log.Println("警告: JWT_SECRET 环境变量未设置，使用默认值")
		jwtSecret = "default_secret_for_development_only"
	}

	if jwtExpireStr == "" {
		log.Println("警告: JWT_EXPIRE 环境变量未设置，使用默认值")
		jwtExpireStr = "24h"
	}

	// 解析过期时间
	expireDuration, err := time.ParseDuration(jwtExpireStr)
	if err != nil {
		return nil, errors.New("JWT过期时间配置错误")
	}

	// 创建 JWT Claims
	claims := &Claims{
		UserID:  user.ID,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// 生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:   tokenString,
		UserID:  user.ID,
		IsAdmin: user.IsAdmin,
		Email:   user.Email, // 新增返回邮箱
	}, nil
}

// Register 处理用户注册
func (s *AuthService) Register(email, password string) (*LoginResult, error) {
	// 创建用户
	user, err := s.authRepo.CreateUser(email, password)
	if err != nil {
		return nil, err
	}

	// 生成 JWT Token
	jwtSecret := os.Getenv("JWT_SECRET")
	jwtExpireStr := os.Getenv("JWT_EXPIRE")

	// 如果环境变量为空，记录警告并使用默认值
	if jwtSecret == "" {
		log.Println("警告: JWT_SECRET 环境变量未设置，使用默认值")
		jwtSecret = "default_secret_for_development_only"
	}

	if jwtExpireStr == "" {
		log.Println("警告: JWT_EXPIRE 环境变量未设置，使用默认值")
		jwtExpireStr = "24h"
	}

	expireDuration, err := time.ParseDuration(jwtExpireStr)
	if err != nil {
		return nil, errors.New("JWT过期时间配置错误")
	}

	claims := &Claims{
		UserID:  user.ID,
		IsAdmin: user.IsAdmin, // 默认为0
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:   tokenString,
		UserID:  user.ID,
		IsAdmin: user.IsAdmin,
		Email:   user.Email, // 新增返回邮箱
	}, nil
}
