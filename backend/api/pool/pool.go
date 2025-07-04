// api/pool/pool.go
package pool

import (
	"log"
	"net/http"
	"portal/pkg/pool"
	"portal/pkg/response"
	"portal/repository"
	"portal/service/instance"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AccountOutput 定义账号信息的输出结构体，确保字段顺序
type AccountOutput struct {
	ID                   string          `json:"id"`                     // 账号ID
	UserID               string          `json:"user_id"`                // 用户ID
	Key1                 string          `json:"key1"`                   // Key1
	Key2                 string          `json:"key2"`                   // Key2
	Email                *string         `json:"email,omitempty"`        // 邮箱
	Password             *string         `json:"password,omitempty"`     // 密码
	Quatos               *string         `json:"quatos,omitempty"`       // 配额
	HK                   *string         `json:"hk,omitempty"`           // HK状态
	VMCount              *int            `json:"vm_count,omitempty"`     // 虚拟机数量
	Region               *string         `json:"region,omitempty"`       // 区域代码
	CreateTime           *string         `json:"create_time,omitempty"`  // 创建时间
	IsSkipped            bool            `json:"is_skipped"`             // 是否跳过
	ErrorNote            string          `json:"error_note"`             // 错误备注
	SkippedInstanceTypes map[string]bool `json:"skipped_instance_types"` // 特定实例类型跳过状态
	RegionUsedCount      int             `json:"region_used_count"`      // 区域已使用的实例计数
}

// PoolInfo 定义账号池信息的输出结构体
type PoolInfo struct {
	Total     int             `json:"total"`     // 总账号数
	Available int             `json:"available"` // 可用账号数
	Accounts  []AccountOutput `json:"accounts"`  // 账号列表
}

// MakeupQueueOutput 补机队列项的输出格式
type MakeupQueueOutput struct {
	UserID         string    `json:"user_id"`         // 用户ID
	Region         string    `json:"region"`          // 区域代码
	RegionDisplay  string    `json:"region_display"`  // 区域显示名称
	TotalCount     int       `json:"total_count"`     // 需要补机总数
	CompletedCount int       `json:"completed_count"` // 已完成数量
	AddTime        time.Time `json:"add_time"`        // 添加到队列的时间
	Status         string    `json:"status"`          // 状态
	Remaining      int       `json:"remaining"`       // 剩余需要补机数量
}

// GetUserInstances 获取当前用户的实例列表
func GetUserInstances(c *gin.Context) {
	// 从 context 获取用户ID（改为获取string类型）
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 直接使用string类型的userID
	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID类型错误")
		return
	}

	// 获取当前用户的实例列表
	instances := pool.GlobalPool.GetInstancesByUserID(userIDStr)

	response.Success(c, http.StatusOK, gin.H{
		"total": len(instances),
		"list":  instances,
	})
}

// GetAllInstances 管理员接口：获取所有实例列表
func GetAllInstances(c *gin.Context) {
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

	// 获取所有实例列表
	instances := pool.GlobalPool.GetAllInstances()

	response.Success(c, http.StatusOK, gin.H{
		"total": len(instances),
		"list":  instances,
	})
}

// GetAccountPool 获取账号池信息（管理员接口）
func GetAccountPool(c *gin.Context) {
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

	// 获取账号池信息
	accountPool := pool.GetAccountPool()

	// 获取所有账号
	accounts := accountPool.GetAllAccounts()

	// 转换为输出结构体数组
	accountOutputs := make([]AccountOutput, 0, len(accounts))
	for _, account := range accounts {
		output := AccountOutput{
			ID:                   account.ID,
			UserID:               account.UserID,
			Key1:                 account.Key1,
			Key2:                 account.Key2,
			IsSkipped:            account.IsSkipped,
			ErrorNote:            account.ErrorNote,
			SkippedInstanceTypes: account.SkippedInstanceTypes,
			RegionUsedCount:      account.RegionUsedCount,
		}

		// 处理可能为空的指针字段
		if account.Email != nil {
			output.Email = account.Email
		}
		if account.Password != nil {
			output.Password = account.Password
		}
		if account.Quatos != nil {
			output.Quatos = account.Quatos
		}
		if account.HK != nil {
			output.HK = account.HK
		}
		if account.VMCount != nil {
			output.VMCount = account.VMCount
		}
		if account.Region != nil {
			output.Region = account.Region
		}
		if account.CreateTime != nil {
			timeStr := account.CreateTime.Format("2006-01-02 15:04:05")
			output.CreateTime = &timeStr
		}

		// 确保 SkippedInstanceTypes 字段被初始化
		if output.SkippedInstanceTypes == nil {
			output.SkippedInstanceTypes = make(map[string]bool)
		}

		accountOutputs = append(accountOutputs, output)
	}

	// 按ID升序排序（数值排序）
	sort.Slice(accountOutputs, func(i, j int) bool {
		idI, errI := strconv.Atoi(accountOutputs[i].ID)
		idJ, errJ := strconv.Atoi(accountOutputs[j].ID)

		// 如果转换成功，则按数字大小排序
		if errI == nil && errJ == nil {
			return idI < idJ
		}

		// 转换失败则按字符串排序
		return accountOutputs[i].ID < accountOutputs[j].ID
	})

	// 构建响应数据
	poolInfo := PoolInfo{
		Total:     len(accountOutputs),
		Available: accountPool.AvailableSize(),
		Accounts:  accountOutputs,
	}

	response.Success(c, http.StatusOK, poolInfo)
}

// 修改 DeleteInstanceItem 结构体，确保 Region 字段属性更明确
type DeleteInstanceItem struct {
	AccountID  string `json:"account_id" binding:"required"`
	Region     string `json:"region" binding:"required"` // 修改为必填字段
	InstanceID string `json:"instance_id" binding:"required"`
}

// DeleteRequest 删除实例请求结构
type DeleteRequest struct {
	Instances []DeleteInstanceItem `json:"instances" binding:"required,min=1"`
}

// DeleteInstance 删除实例接口
func DeleteInstance(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误:"+err.Error())
		return
	}

	// 从context获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 转换为string类型
	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID类型错误")
		return
	}

	// 转换请求参数到服务层的类型
	serviceInstances := make([]instance.DeleteInstanceItem, len(req.Instances))
	for i, item := range req.Instances {
		// 处理区域参数，支持中文和英文简写
		region := item.Region
		if region != "" {
			switch region {
			case "jp", "japan", "日本":
				region = "ap-northeast-3" // 日本区域
			case "sg", "singapore", "新加坡":
				region = "ap-southeast-1" // 新加坡区域
			case "hk", "hongkong", "香港":
				region = "ap-east-1" // 香港区域
			}
		}
		// 不再设置默认区域，让服务层根据账号信息决定

		serviceInstances[i] = instance.DeleteInstanceItem{
			AccountID:  item.AccountID,
			Region:     region,
			InstanceID: item.InstanceID,
		}
	}

	svc := instance.NewInstanceService(repository.GetDB())
	results, err := svc.Delete(c, userIDStr, serviceInstances)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "删除实例失败:"+err.Error())
		return
	}

	response.Success(c, http.StatusOK, results)
}

// 修改 ChangeIPItem 结构体，确保 Region 字段属性更明确
type ChangeIPItem struct {
	AccountID  string `json:"account_id" binding:"required"`
	Region     string `json:"region" binding:"required"` // 修改为必填字段
	InstanceID string `json:"instance_id" binding:"required"`
}

// ChangeIPRequest 更换IP请求结构
type ChangeIPRequest struct {
	Instances []ChangeIPItem `json:"instances" binding:"required,min=1"`
}

// ChangeIP 更换IP接口
func ChangeIP(c *gin.Context) {
	var req ChangeIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误:"+err.Error())
		return
	}

	// 从context获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 转换为string类型
	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID类型错误")
		return
	}

	// 转换请求参数到服务层的类型
	serviceInstances := make([]instance.ChangeIPItem, len(req.Instances))
	for i, item := range req.Instances {
		// 处理区域参数，支持中文和英文简写
		region := item.Region
		if region != "" {
			switch region {
			case "jp", "japan", "日本":
				region = "ap-northeast-3" // 日本区域
			case "sg", "singapore", "新加坡":
				region = "ap-southeast-1" // 新加坡区域
			case "hk", "hongkong", "香港":
				region = "ap-east-1" // 香港区域
			}
		}
		// 不再设置默认区域，让服务层根据账号信息决定

		serviceInstances[i] = instance.ChangeIPItem{
			AccountID:  item.AccountID,
			Region:     region,
			InstanceID: item.InstanceID,
		}
	}

	svc := instance.NewInstanceService(repository.GetDB())
	results, err := svc.ChangeIP(c, userIDStr, serviceInstances)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更换IP失败:"+err.Error())
		return
	}

	// 更换IP成功后，触发IP变更事件
	eventManager := pool.GetEventManager()
	for _, result := range results {
		if result.Status == "成功" && result.NewIP != "" {
			// 触发IP变更事件
			eventManager.TriggerIPChangeEvent(result.InstanceID, result.NewIP)
		}
	}

	response.Success(c, http.StatusOK, results)
}

// ResetAccountsRequest 重置账号请求结构
type ResetAccountsRequest struct {
	AccountIDs []string `json:"account_ids" binding:"required,min=1"`
}

// ResetAccounts 重置指定账号的状态（管理员接口）
func ResetAccounts(c *gin.Context) {
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

	var req ResetAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误:"+err.Error())
		return
	}

	// 获取账号池实例
	accountPool := pool.GetAccountPool()

	// 重置指定账号的状态
	successIDs := make([]string, 0)
	failedIDs := make([]string, 0)

	for _, accountID := range req.AccountIDs {
		// 获取账号信息，确认账号存在
		account := accountPool.GetAccount(accountID)
		if account == nil {
			failedIDs = append(failedIDs, accountID)
			continue
		}

		// 重置账号状态
		accountPool.ResetAccountStatus(accountID)

		// 验证重置是否成功
		updatedAccount := accountPool.GetAccount(accountID)
		if updatedAccount != nil {
			// 确认状态已被重置
			if !updatedAccount.IsSkipped &&
				len(updatedAccount.SkippedInstanceTypes) == 0 &&
				updatedAccount.ErrorNote == "" {
				successIDs = append(successIDs, accountID)
			} else {
				// 重置未完全成功
				failedIDs = append(failedIDs, accountID)
			}
		} else {
			failedIDs = append(failedIDs, accountID)
		}
	}

	// 构建响应
	response.Success(c, http.StatusOK, gin.H{
		"success": gin.H{
			"count": len(successIDs),
			"ids":   successIDs,
		},
		"failed": gin.H{
			"count": len(failedIDs),
			"ids":   failedIDs,
		},
	})
}

// GetMakeupQueue 获取补机队列信息（管理员接口）
func GetMakeupQueue(c *gin.Context) {
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

	// 获取补机队列
	makeupQueue := pool.GetMakeupQueue()
	queueItems := makeupQueue.GetQueue()

	// 转换为输出结构
	outputs := make([]MakeupQueueOutput, 0, len(queueItems))
	for _, item := range queueItems {
		// 确保显示正确的区域名称
		region := item.Region
		var regionDisplay string

		switch region {
		case "ap-northeast-3":
			regionDisplay = "日本区"
		case "ap-southeast-1":
			regionDisplay = "新加坡区"
		case "ap-east-1", "":
			region = "ap-east-1" // 确保默认值一致
			regionDisplay = "香港区"
		default:
			regionDisplay = region // 未知区域则直接显示代码
		}

		output := MakeupQueueOutput{
			UserID:         item.UserID,
			Region:         region,
			RegionDisplay:  regionDisplay, // 添加显示名称
			TotalCount:     item.TotalCount,
			CompletedCount: item.CompletedCount,
			AddTime:        item.AddTime,
			Status:         item.Status,
			Remaining:      item.TotalCount - item.CompletedCount,
		}
		outputs = append(outputs, output)
	}

	// 获取队列状态摘要
	status := makeupQueue.GetQueueStatus()

	// 构建响应
	response.Success(c, http.StatusOK, gin.H{
		"queue":  outputs,
		"status": status,
	})
}

// ResetMakeupQueue 重置卡住的补机队列（管理员接口）
func ResetMakeupQueue(c *gin.Context) {
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

	// 将 interface{} 转换为 uint8，然后与 1 比较
	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 获取补机队列并重置卡住的任务
	makeupQueue := pool.GetMakeupQueue()
	makeupQueue.ResetStuckTasks() // 确保此方法能处理所有区域的任务
	log.Printf("管理员[%s]已重置所有区域的卡住补机任务", userID)

	// 触发手动重置事件
	pool.GetEventManager().TriggerEvent(pool.ManualReset, "")

	// 构建响应
	response.Success(c, http.StatusOK, gin.H{
		"message": "已重置所有区域的卡住补机任务",
	})
}

// ClearMakeupQueue 清空所有补机队列（管理员接口）
func ClearMakeupQueue(c *gin.Context) {
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

	// 将 interface{} 转换为 uint8，然后与 1 比较
	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return
	}

	// 获取补机队列并清空
	makeupQueue := pool.GetMakeupQueue()
	makeupQueue.ClearAllQueue() // 确保此方法能处理所有区域的队列
	log.Printf("管理员[%s]已清空所有区域的补机队列", userID)

	// 构建响应
	response.Success(c, http.StatusOK, gin.H{
		"message": "已清空所有区域的补机队列",
	})
}
