// routes/auth.go
package routes

import (
	"portal/api/auth"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes 注册认证相关路由
func RegisterAuthRoutes(router *gin.Engine) {
	router.POST("/login", auth.Login)
	router.POST("/register", auth.Register)
}
