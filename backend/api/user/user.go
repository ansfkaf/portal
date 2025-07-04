// api/user/user.go
package user

import (
	"errors"
	"fmt"
	"net/http"
	"portal/model"
	"portal/pkg/pool"
	"portal/pkg/response"
	"portal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUsersRequest 获取用户信息请求
type GetUsersRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

// UpdateUsersRequest 更新用户信息请求
type UpdateUsersRequest struct {
	IDs      []string `json:"ids" binding:"required"`
	Password string   `json:"password"`
	Email    string   `json:"email"`
	IsAdmin  *uint8   `json:"is_admin"`
}

// checkAdminPermission 检查管理员权限
func checkAdminPermission(c *gin.Context) bool {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return false
	}

	isAdmin, exists := c.Get("is_admin")
	if !exists {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return false
	}

	if adminValue, ok := isAdmin.(uint8); !ok || adminValue != 1 {
		response.Error(c, http.StatusForbidden, "需要管理员权限")
		return false
	}

	return true
}

// GetUsers 获取指定ID的用户信息
// GetUsers 获取用户信息
func GetUsers(c *gin.Context) {
	// 验证管理员权限
	if !checkAdminPermission(c) {
		return
	}

	var req GetUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有提供IDs或解析JSON失败，则获取所有用户
		allUsers, err := model.GetAllUsers(repository.GetDB())
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "获取用户信息失败: "+err.Error())
			return
		}

		response.Success(c, http.StatusOK, gin.H{
			"users": allUsers,
			"total": len(allUsers),
		})
		return
	}

	// 如果提供了IDs，则获取指定用户
	users, err := model.GetUsersByIDs(repository.GetDB(), req.IDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取用户信息失败: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"users": users,
		"total": len(users),
	})
}

// UpdateUsers 更新用户信息
func UpdateUsers(c *gin.Context) {
	// 验证管理员权限
	if !checkAdminPermission(c) {
		return
	}

	var req UpdateUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// 调试代码：打印接收到的密码
	fmt.Printf("接收到的密码: '%s'，长度: %d\n", req.Password, len(req.Password))

	// 准备更新数据
	updateData := make(map[string]interface{})
	updateData["ids"] = req.IDs

	// 检查是否提供了密码
	if req.Password != "" {
		updateData["password"] = req.Password
	}
	// 检查是否提供了邮箱
	if req.Email != "" {
		updateData["email"] = req.Email
	}

	// 检查是否提供了管理员状态
	if req.IsAdmin != nil {
		updateData["is_admin"] = *req.IsAdmin
	}

	// 执行更新
	err := model.UpdateUsers(repository.GetDB(), updateData)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新用户信息失败: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "用户信息更新成功")
}

// MakeupUsersRequest 用户补机请求
type MakeupUsersRequest struct {
	IDs    []string `json:"ids" binding:"required"`
	Region string   `json:"region"` // 可选参数：区域代码
	Count  int      `json:"count"`  // 可选参数：补机数量
}

// MakeupUsers 为指定用户执行补机操作
func MakeupUsers(c *gin.Context) {
	// 验证管理员权限
	if !checkAdminPermission(c) {
		return
	}

	var req MakeupUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// 获取补机队列
	makeupQueue := pool.GetMakeupQueue()

	// 默认补机数量为1
	count := req.Count
	if count <= 0 {
		count = 1
	}

	// 补机结果
	successCount := 0
	failedIDs := make([]string, 0)

	// 处理每个用户
	for _, userID := range req.IDs {
		// 确定区域：如果请求中有区域参数则使用，否则从用户设置中获取
		region := req.Region
		if region == "" {
			// 从用户设置中获取默认区域
			setting, err := model.GetSettingByUserID(repository.GetDB(), userID)
			if err != nil {
				failedIDs = append(failedIDs, userID)
				fmt.Printf("获取用户[%s]设置失败: %v\n", userID, err)
				continue
			}
			region = setting.GetRegionCode()
			fmt.Printf("获取用户[%s]默认区域: %s\n", userID, region)
		}

		// 管理员手动补机时，直接添加到补机队列 - 已移除额外的布尔参数
		makeupQueue.AddToQueueWithRegion(userID, count, region)
		successCount++
	}

	response.Success(c, http.StatusOK, gin.H{
		"message":       fmt.Sprintf("已提交%d个用户的补机任务", successCount),
		"success_count": successCount,
		"failed_ids":    failedIDs,
	})
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	IsAdmin  uint8  `json:"is_admin"`
}

// CreateUser 创建新用户
func CreateUser(c *gin.Context) {
	// 验证管理员权限
	if !checkAdminPermission(c) {
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 先检查是否存在相同邮箱的用户
	var existingUser model.User
	result := repository.GetDB().Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		// 找到了用户，说明邮箱已被使用
		response.Error(c, http.StatusConflict, fmt.Sprintf("用户 %s 已存在", req.Email))
		return
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// 如果发生了其他错误（不是未找到记录的错误）
		response.Error(c, http.StatusInternalServerError, "检查用户是否存在时发生错误: "+result.Error.Error())
		return
	}

	// 创建用户
	user, err := model.CreateUser(repository.GetDB(), req.Email, req.Password, req.IsAdmin)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "创建用户失败: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{
		"message": "用户创建成功",
		"user": map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}
