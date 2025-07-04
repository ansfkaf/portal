package middleware

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 自定义JWT的声明
type CustomClaims struct {
	UserID  string `json:"user_id"` // 修改为string类型
	IsAdmin uint8  `json:"is_admin"`
	jwt.RegisteredClaims
}

var jwtSecret []byte
var jwtExpire time.Duration

// 移除init函数，改为延迟初始化

// 获取JWT Secret
func getJWTSecret() []byte {
	if len(jwtSecret) == 0 {
		// 延迟初始化，第一次使用时加载
		secretStr := os.Getenv("JWT_SECRET")
		if secretStr == "" {
			// 如果环境变量未设置，使用默认值并记录日志
			fmt.Println("警告: JWT_SECRET 环境变量未设置，使用默认值")
			secretStr = "default_jwt_secret_for_development"
		}
		jwtSecret = []byte(secretStr)
	}
	return jwtSecret
}

// 获取JWT过期时间
func getJWTExpire() time.Duration {
	if jwtExpire == 0 {
		// 延迟初始化，第一次使用时加载
		expireStr := os.Getenv("JWT_EXPIRE")
		if expireStr == "" {
			// 如果环境变量未设置，使用默认值并记录日志
			fmt.Println("警告: JWT_EXPIRE 环境变量未设置，使用默认值 24h")
			expireStr = "24h"
		}

		var err error
		jwtExpire, err = time.ParseDuration(expireStr)
		if err != nil {
			// 如果解析失败，使用默认值
			fmt.Printf("警告: 解析JWT过期时间失败: %v，使用默认值 24h\n", err)
			jwtExpire = 24 * time.Hour
		}
	}
	return jwtExpire
}

// GenerateToken 生成JWT token
func GenerateToken(userID string, isAdmin uint8) (string, error) { // 修改参数类型为string
	claims := CustomClaims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(getJWTExpire())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ParseToken 解析JWT token
func ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"code": 401,
				"msg":  "Authorization header is required",
			})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(401, gin.H{
				"code": 401,
				"msg":  "Authorization header format must be Bearer {token}",
			})
			c.Abort()
			return
		}

		// 解析token
		claims, err := ParseToken(parts[1])
		if err != nil {
			c.JSON(401, gin.H{
				"code":  401,
				"msg":   "Invalid or expired token",
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("is_admin", claims.IsAdmin)
		c.Next()
	}
}
