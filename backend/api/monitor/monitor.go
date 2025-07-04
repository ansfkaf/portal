// api/monitor/monitor.go
package monitor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"portal/model"
	"portal/pkg/pool"
	"portal/pkg/response"
	"portal/pkg/tg"
	"portal/repository"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UpdateConfigRequest 更新配置请求结构
type UpdateConfigRequest struct {
	Threshold        int    `json:"threshold"`           // 香港区阈值
	JpThreshold      int    `json:"jp_threshold"`        // 日本区阈值
	SgThreshold      int    `json:"sg_threshold"`        // 新加坡区阈值
	IsEnabled        bool   `json:"is_enabled"`          // 开关状态
	IsTgEnabled      bool   `json:"is_tg_enabled"`       // TG通知开关
	TgUserID         string `json:"tg_user_id"`          // TG用户ID
	IsIPRangeEnabled bool   `json:"is_ip_range_enabled"` // IP段限制开关
	IPRange          string `json:"ip_range"`            // 香港IP段
	JpIPRange        string `json:"jp_ip_range"`         // 日本IP段
	SgIPRange        string `json:"sg_ip_range"`         // 新加坡IP段
}

// AdminUpdateConfigRequest 管理员更新配置请求结构
type AdminUpdateConfigRequest struct {
	UserID           string `json:"user_id"`             // 要更新的用户ID
	Threshold        int    `json:"threshold"`           // 香港区阈值
	JpThreshold      int    `json:"jp_threshold"`        // 日本区阈值
	SgThreshold      int    `json:"sg_threshold"`        // 新加坡区阈值
	IsEnabled        bool   `json:"is_enabled"`          // 开关状态
	IsTgEnabled      bool   `json:"is_tg_enabled"`       // TG通知开关
	TgUserID         string `json:"tg_user_id"`          // TG用户ID
	IsIPRangeEnabled bool   `json:"is_ip_range_enabled"` // IP段限制开关
	IPRange          string `json:"ip_range"`            // 香港IP段
	JpIPRange        string `json:"jp_ip_range"`         // 日本IP段
	SgIPRange        string `json:"sg_ip_range"`         // 新加坡IP段
}

// MakeupHistoryRecord 补机历史记录响应结构 (修改后)
type MakeupHistoryRecord struct {
	UserID    string    `json:"user_id"`   // 用户ID
	Region    string    `json:"region"`    // 区域代码
	Count     int       `json:"count"`     // 补机数量
	Timestamp time.Time `json:"timestamp"` // 补机时间
}

// BindingResponse 绑定码响应结构
type BindingResponse struct {
	BindingCode string `json:"binding_code"` // 绑定码
	BindingURL  string `json:"binding_url"`  // 绑定URL
}

// GetUserConfig 获取当前用户的监控配置
func GetUserConfig(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 获取用户的监控配置
	config, err := model.GetMonitorByUserID(repository.GetDB(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取监控配置失败")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"config": config,
	})
}

// UpdateUserConfig 更新当前用户的监控配置
func UpdateUserConfig(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 解析请求体
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 首先获取当前用户的配置信息
	currentConfig, err := model.GetMonitorByUserID(repository.GetDB(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取当前配置失败")
		return
	}

	// 检查用户是否是管理员
	isAdmin, exists := c.Get("is_admin")
	isAdminUser := exists && isAdmin.(uint8) == 1

	// 确定要更新的阈值
	threshold := currentConfig.Threshold
	jpThreshold := currentConfig.JpThreshold
	sgThreshold := currentConfig.SgThreshold

	if isAdminUser {
		// 管理员可以更新阈值
		threshold = req.Threshold
		jpThreshold = req.JpThreshold
		sgThreshold = req.SgThreshold
	}
	// 非管理员无法修改阈值，保持原值

	// 更新监控基础配置
	err = model.UpdateMonitor(repository.GetDB(), userID, threshold, jpThreshold, sgThreshold, req.IsEnabled)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新监控配置失败")
		return
	}

	// 更新TG通知设置
	// 只有管理员可以修改TG用户ID，普通用户保持当前ID
	tgUserID := currentConfig.TgUserID
	if isAdminUser && req.TgUserID != "" {
		tgUserID = req.TgUserID
	}

	err = model.UpdateTgSettings(repository.GetDB(), userID, req.IsTgEnabled, tgUserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新TG通知设置失败")
		return
	}

	// 更新多区域IP段限制设置
	// 判断是否提供了日本和新加坡的IP范围
	jpIPRange := req.JpIPRange
	sgIPRange := req.SgIPRange

	// 更新所有区域的IP段限制设置
	err = model.UpdateAllIPRangeSettings(repository.GetDB(), userID, req.IsIPRangeEnabled, req.IPRange, jpIPRange, sgIPRange)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新IP段限制设置失败")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

// GetAllConfigs 管理员接口：获取所有用户的监控配置
func GetAllConfigs(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 验证管理员权限
	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 将 interface{} 转换为 uint8，然后与 1 比较
	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 获取所有用户的监控配置
	configs, err := model.GetAllMonitors(repository.GetDB())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取监控配置失败")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"total": len(configs),
		"list":  configs,
	})
}

// AdminUpdateConfig 管理员更新指定用户的监控配置
func AdminUpdateConfig(c *gin.Context) {
	// 验证管理员权限
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 解析请求体
	var req AdminUpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证要更新的用户是否存在
	if req.UserID == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	// 更新监控基础配置
	err := model.UpdateMonitor(repository.GetDB(), req.UserID, req.Threshold, req.JpThreshold, req.SgThreshold, req.IsEnabled)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新监控配置失败")
		return
	}

	// 更新TG通知设置（管理员可以直接修改TG用户ID）
	err = model.UpdateTgSettings(repository.GetDB(), req.UserID, req.IsTgEnabled, req.TgUserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新TG通知设置失败")
		return
	}

	// 更新所有区域的IP段限制设置
	err = model.UpdateAllIPRangeSettings(repository.GetDB(), req.UserID, req.IsIPRangeEnabled, req.IPRange, req.JpIPRange, req.SgIPRange)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新IP段限制设置失败")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

// GetMakeupHistory 获取补机历史记录
func GetMakeupHistory(c *gin.Context) {
	// 验证管理员权限
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 获取补机历史记录
	rawHistory := pool.GlobalMakeupHistory.GetAllRecords()

	// 转换为列表格式并按时间排序
	var historyList []MakeupHistoryRecord
	for combinedID, records := range rawHistory {
		// 解析组合ID (格式: "用户ID:区域" 或 "用户ID")
		userIDPart := combinedID
		regionPart := "ap-east-1" // 默认为香港区域

		if strings.Contains(combinedID, ":") {
			parts := strings.Split(combinedID, ":")
			if len(parts) == 2 {
				userIDPart = parts[0]
				regionPart = parts[1]
			}
		}

		// 区域代码转换为用户友好的显示值
		regionDisplay := regionPart
		switch regionPart {
		case "ap-east-1":
			regionDisplay = "ap-east-1" // 香港区域
		case "ap-northeast-3":
			regionDisplay = "ap-northeast-3" // 日本区域
		case "ap-southeast-1":
			regionDisplay = "ap-southeast-1" // 新加坡区域
		}

		for _, record := range records {
			historyList = append(historyList, MakeupHistoryRecord{
				UserID:    userIDPart,
				Region:    regionDisplay,
				Count:     record.Count,
				Timestamp: record.Timestamp,
			})
		}
	}

	// 按时间倒序排序
	sort.Slice(historyList, func(i, j int) bool {
		return historyList[i].Timestamp.After(historyList[j].Timestamp)
	})

	response.Success(c, http.StatusOK, gin.H{
		"total": len(historyList),
		"list":  historyList,
	})
}

// GenerateTgBindingCode 生成TG绑定码
func GenerateTgBindingCode(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 获取TG客户端
	tgClient, err := tg.GetClient()
	if err != nil {
		// 如果客户端未初始化，则尝试初始化
		if err := tg.InitTgClient(); err != nil {
			response.Error(c, http.StatusInternalServerError, "TG客户端初始化失败: "+err.Error())
			return
		}
		tgClient, err = tg.GetClient()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "获取TG客户端失败: "+err.Error())
			return
		}
	}

	// 生成绑定码
	bindingCode, err := tgClient.CreateBindingCode(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成绑定码失败: "+err.Error())
		return
	}

	// 获取Bot信息
	botInfo := tgClient.GetBotInfo()
	bindingURL := fmt.Sprintf("/bind/%s", bindingCode)

	response.Success(c, http.StatusOK, gin.H{
		"binding_code": bindingCode,
		"binding_url":  bindingURL,
		"bot_username": "@" + botInfo.UserName,
	})
}

// UnbindTgUser 解绑TG用户
func UnbindTgUser(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 解绑用户的TG账号
	err := model.UnbindTgUser(repository.GetDB(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "解绑TG账号失败: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "解绑成功",
	})
}

// TriggerDetection 立即触发主动检测
func TriggerDetection(c *gin.Context) {
	// 验证管理员权限
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 获取全局检测器实例
	detector := pool.GlobalDetector
	if detector == nil {
		response.Error(c, http.StatusInternalServerError, "检测器未初始化")
		return
	}

	// 触发所有用户的主动检测
	log.Printf("管理员[%s]触发所有用户的主动检测", userID)
	results := detector.DetectAllUsers()

	if len(results) > 0 {
		response.Success(c, http.StatusOK, gin.H{
			"message": "检测完成，发现需要补机的用户",
			"results": results,
		})
	} else {
		response.Success(c, http.StatusOK, gin.H{
			"message": "检测完成，所有用户无需补机",
			"results": []pool.DetectResult{}, // 使用完全限定的类型名
		})
	}
}

// ClearHistory 清空补机历史和冷却状态
func ClearHistory(c *gin.Context) {
	// 验证管理员权限
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 获取全局补机历史记录实例
	makeupHistory := pool.GlobalMakeupHistory
	if makeupHistory == nil {
		response.Error(c, http.StatusInternalServerError, "补机历史记录管理器未初始化")
		return
	}

	// 获取账号池实例
	accountPool := pool.GetAccountPool()
	if accountPool == nil {
		response.Error(c, http.StatusInternalServerError, "账号池未初始化")
		return
	}

	// 清空所有补机历史记录
	makeupHistory.ClearAllRecords() // 确保此方法能处理所有区域的记录
	log.Printf("管理员[%s]已清空所有区域的补机历史记录", userID)

	// 重置所有被标记为跳过的账号
	accountPool.ResetAllAccountsStatus() // 确保此方法重置所有区域的账号状态
	log.Printf("管理员[%s]已重置所有区域的账号冷却状态", userID)

	response.Success(c, http.StatusOK, gin.H{
		"message": "已清空所有区域的补机历史记录并重置账号冷却状态",
	})
}

// BackupMonitorSettings 备份所有监控配置
func BackupMonitorSettings(c *gin.Context) {
	// 验证管理员权限
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 检查临时备份表是否已存在
	db := repository.GetDB()
	if db.Migrator().HasTable("monitor_backup") {
		// 如果已存在，先删除它
		if err := db.Migrator().DropTable("monitor_backup"); err != nil {
			response.Error(c, http.StatusInternalServerError, "删除已存在的备份表失败: "+err.Error())
			return
		}
	}

	// 创建临时备份表 (与monitor表结构相同)
	if err := db.Exec("CREATE TABLE monitor_backup LIKE monitor").Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建备份表结构失败: "+err.Error())
		return
	}

	// 将数据从monitor表复制到monitor_backup表
	if err := db.Exec("INSERT INTO monitor_backup SELECT * FROM monitor").Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "备份数据失败: "+err.Error())
		return
	}

	// 关闭所有用户的TG通知
	if err := db.Exec("UPDATE monitor SET is_tg_enabled = false").Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新TG通知状态失败: "+err.Error())
		return
	}

	log.Printf("管理员[%s]已备份所有监控配置并临时关闭所有TG通知", userID)

	response.Success(c, http.StatusOK, gin.H{
		"message": "已成功备份所有监控配置并临时关闭所有TG通知",
	})
}

// RestoreMonitorSettings 从备份恢复所有监控配置
func RestoreMonitorSettings(c *gin.Context) {
	// 验证管理员权限
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	db := repository.GetDB()

	// 检查临时备份表是否存在
	if !db.Migrator().HasTable("monitor_backup") {
		response.Error(c, http.StatusBadRequest, "备份表不存在，无法恢复数据")
		return
	}

	// 从备份表恢复数据 - 先清空当前表
	if err := db.Exec("DELETE FROM monitor").Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "清空当前数据失败: "+err.Error())
		return
	}

	// 从备份表恢复数据
	if err := db.Exec("INSERT INTO monitor SELECT * FROM monitor_backup").Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "恢复数据失败: "+err.Error())
		return
	}

	// 删除临时备份表
	if err := db.Migrator().DropTable("monitor_backup"); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除备份表失败: "+err.Error())
		return
	}

	log.Printf("管理员[%s]已恢复所有监控配置", userID)

	response.Success(c, http.StatusOK, gin.H{
		"message": "已成功恢复所有监控配置",
	})
}

// TriggerUserIPRangeCheck 普通用户触发自己的IP范围检查
func TriggerUserIPRangeCheck(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 获取全局IP范围检查器实例
	ipChecker := pool.GlobalIPChecker
	if ipChecker == nil {
		response.Error(c, http.StatusInternalServerError, "IP范围检查器未初始化")
		return
	}

	log.Printf("用户[%s]触发自己的IP范围检查", userID)
	go func() {
		ctx := context.Background()
		if err := pool.TriggerIPRangeCheck(ctx, userID); err != nil {
			log.Printf("触发用户[%s]的IP范围检查失败: %v", userID, err)
		}
	}()

	response.Success(c, http.StatusOK, gin.H{
		"message": "已触发IP范围检查，请稍后查看结果",
	})
}

// TriggerAdminIPRangeCheck 管理员触发所有用户的IP范围检查
func TriggerAdminIPRangeCheck(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 验证管理员权限
	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 获取全局IP范围检查器实例
	ipChecker := pool.GlobalIPChecker
	if ipChecker == nil {
		response.Error(c, http.StatusInternalServerError, "IP范围检查器未初始化")
		return
	}

	log.Printf("管理员[%s]触发所有用户的IP范围检查", userID)
	go func() {
		ctx := context.Background()
		if err := pool.TriggerAllIPRangeCheck(ctx); err != nil {
			log.Printf("触发所有用户IP范围检查失败: %v", err)
		}
	}()

	response.Success(c, http.StatusOK, gin.H{
		"message": "已触发所有用户的IP范围检查，请稍后查看结果",
	})
}
