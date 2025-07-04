// routes/dash.go
package routes

import (
	"portal/api/account"
	"portal/api/batchimport"
	"portal/api/instance"
	"portal/api/monitor"
	"portal/api/pool"
	"portal/api/setting"
	"portal/api/user"
	"portal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterDashRoutes(router *gin.Engine) {
	authRequired := router.Group("")
	authRequired.Use(middleware.JWTAuthMiddleware())
	{
		// 导入相关路由
		authRequired.POST("/import", batchimport.ImportAccounts)

		// 设置相关路由
		authRequired.GET("/setting", setting.GetSetting)
		authRequired.POST("/setting", setting.UpdateSetting)
		authRequired.POST("/setting/admin", setting.GetAllSettings)            // 新增: 管理员获取所有设置
		authRequired.POST("/setting/admin/update", setting.AdminUpdateSetting) // 新增: 管理员更新指定用户设置

		// 账号管理路由组
		accountGroup := authRequired.Group("/account")
		{
			accountGroup.GET("/list", account.List)
			accountGroup.POST("/delete", account.Delete)
			accountGroup.POST("/check", account.Check)
			accountGroup.POST("/apply-hk", account.ApplyHK)
			accountGroup.POST("/create-instance", account.CreateInstance) // 创建实例保留在account组
			accountGroup.POST("/clean-t3-micro", account.CleanT3Micro)    // 新增: 清理t3.micro实例
		}

		// 实例管理路由组 - 只包含实例本身的操作
		instanceGroup := authRequired.Group("/instance")
		{
			instanceGroup.POST("/delete", instance.Delete)            // 只移动删除实例到这里
			instanceGroup.POST("/change-ip", instance.ChangeIP)       // 新增更换IP路由
			instanceGroup.GET("/account_list", instance.ListAccounts) // 新增账号列表路由
			instanceGroup.POST("/list", instance.ListInstances)       // 新增实例列表路由
		}

		// 实例池路由组
		poolGroup := authRequired.Group("/pool")
		{
			poolGroup.POST("/admin", pool.GetAllInstances)     // 管理员接口，路径为 /pool/admin
			poolGroup.POST("", pool.GetUserInstances)          // 普通用户接口，路径为 /pool
			poolGroup.GET("/accountpool", pool.GetAccountPool) // 获取账号池信息
			poolGroup.POST("/delete", pool.DeleteInstance)     // 新增: 删除实例接口
			poolGroup.POST("/change-ip", pool.ChangeIP)        // 新增: 更换IP接口
			poolGroup.POST("/reset-accounts", pool.ResetAccounts)
			poolGroup.GET("/makeup-queue", pool.GetMakeupQueue)    // 获取补机队列接口
			poolGroup.POST("/reset-makeup", pool.ResetMakeupQueue) // 重置补机队列
			poolGroup.POST("/clear-makeup", pool.ClearMakeupQueue) // 新增: 清空补机队列
		}

		// 监控路由组
		monitorGroup := authRequired.Group("/monitor")
		{
			monitorGroup.POST("/admin", monitor.GetAllConfigs)                     // 管理员获取所有配置
			monitorGroup.POST("/admin/update", monitor.AdminUpdateConfig)          // 管理员更新指定用户配置
			monitorGroup.GET("", monitor.GetUserConfig)                            // 获取用户配置
			monitorGroup.POST("", monitor.UpdateUserConfig)                        // 更新用户配置
			monitorGroup.POST("/makeup-history", monitor.GetMakeupHistory)         // 获取补机历史记录
			monitorGroup.GET("/tg/bind", monitor.GenerateTgBindingCode)            // 生成TG绑定码
			monitorGroup.POST("/tg/unbind", monitor.UnbindTgUser)                  // 解绑TG账号
			monitorGroup.POST("/admin/clear", monitor.ClearHistory)                // 新增: 清空补机历史和冷却状态
			monitorGroup.POST("/admin/detect", monitor.TriggerDetection)           // 新增: 立即触发主动检测
			monitorGroup.POST("/admin/backup", monitor.BackupMonitorSettings)      // 新增: 备份TG通知设置
			monitorGroup.POST("/admin/restore", monitor.RestoreMonitorSettings)    // 新增: 恢复TG通知设置
			monitorGroup.POST("/check-ip", monitor.TriggerUserIPRangeCheck)        // 新增: 普通用户触发IP范围检查
			monitorGroup.POST("/admin/check-ip", monitor.TriggerAdminIPRangeCheck) // 新增: 管理员触发所有用户的IP范围检查
		}

		// 用户管理路由组
		userGroup := authRequired.Group("/user")
		{
			userGroup.POST("/get", user.GetUsers)       // 获取用户信息
			userGroup.POST("/update", user.UpdateUsers) // 更新用户信息
			userGroup.POST("/makeup", user.MakeupUsers) // 新增: 用户补机
			userGroup.POST("/create", user.CreateUser)  // 新增: 创建用户
		}
	}
}
