// main.go
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"portal/middleware"
	_ "portal/pkg/logger" // 导入日志模块，自动初始化
	"portal/pkg/pool"
	"portal/pkg/tg"
	"portal/repository"
	"portal/routes"
	"portal/utils/s3"
)

func init() {
	// 先检查系统环境变量
	wsURL := os.Getenv("WS_URL")
	if wsURL == "" {
		// 只有在系统环境变量不存在时才加载.env文件
		currentDir, _ := os.Getwd()
		parentDir := filepath.Dir(currentDir)
		envPath := filepath.Join(parentDir, ".env")

		_ = godotenv.Load(envPath)
		wsURL = os.Getenv("WS_URL") // 重新获取
	}

	log.Printf("当前WebSocket域名: %s", wsURL)
}

// setupRouter 配置路由
func setupRouter() *gin.Engine {
	// 禁用 Gin 的日志颜色
	gin.DisableConsoleColor()

	// 设置 gin 运行模式
	gin.SetMode(gin.ReleaseMode)

	// 创建 gin 实例
	r := gin.Default()

	// 使用全局中间件
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.CORSMiddleware())

	// 健康检查路由
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "服务运行正常",
		})
	})

	// WebSocket路由
	r.GET("/ws", pool.HandleWebSocket)

	// 注册路由组
	routes.RegisterAuthRoutes(r) // 注册认证路由
	routes.RegisterDashRoutes(r) // 注册仪表盘相关路由

	// 初始化备份功能
	s3.InitBackup(r)
	return r
}

func main() {
	// 初始化数据库连接
	if err := repository.InitDB(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 初始化连接池服务
	pool.InitPool()

	// 初始化TG客户端
	if err := tg.InitTgClient(); err != nil {
		log.Printf("TG客户端初始化失败: %v", err)
		// 继续运行，不影响主程序
	} else {
		log.Printf("TG客户端初始化成功")
	}

	// 设置默认端口为 8080
	port := "8080"

	// 获取路由
	r := setupRouter()

	// 启动服务器
	serverAddr := "0.0.0.0:8080" // 修改这里，明确监听所有地址
	log.Printf("服务器启动于端口 %s", port)

	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
