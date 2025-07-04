// api/auth/auth.go
package auth

import (
	"portal/pkg/response"
	"portal/repository"
	"portal/service/auth"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Email    string `json:"email"` // 修改为email
	Password string `json:"password"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	UserID  string `json:"user_id"`
	IsAdmin uint8  `json:"is_admin"`
	Email   string `json:"email"` // 新增返回邮箱字段
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login 处理登录请求
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求体格式无效")
		return
	}

	// 使用数据库连接初始化 AuthService
	authService := auth.NewAuthService(repository.GetDB())
	loginResult, err := authService.Login(req.Email, req.Password)
	if err != nil {
		response.Error(c, 401, err.Error())
		return
	}

	response.Success(c, 200, LoginResponse{
		Token:   loginResult.Token,
		UserID:  loginResult.UserID,
		IsAdmin: loginResult.IsAdmin,
		Email:   loginResult.Email, // 新增返回邮箱
	})
}

// Register 处理注册请求
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求体格式无效")
		return
	}

	authService := auth.NewAuthService(repository.GetDB())
	registerResult, err := authService.Register(req.Email, req.Password)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, 200, LoginResponse{
		Token:   registerResult.Token,
		UserID:  registerResult.UserID,
		IsAdmin: registerResult.IsAdmin,
		Email:   registerResult.Email, // 新增返回邮箱
	})
}
