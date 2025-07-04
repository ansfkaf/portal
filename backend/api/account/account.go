// api/account/account.go
package account

import (
	"net/http"
	"portal/pkg/response"
	"portal/repository"
	"portal/service/account"

	"github.com/gin-gonic/gin"
)

// DeleteRequest 删除账号请求结构
type DeleteRequest struct {
	AccountIDs []string `json:"account_ids" binding:"required,min=1"`
}

// List 获取账号列表
func List(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	accountService := account.NewAccountService(repository.GetDB())
	// 使用用户ID获取账号列表
	result, err := accountService.List(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, result)
}

// Delete 删除账号
func Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效:"+err.Error())
		return
	}

	// 添加参数验证
	if len(req.AccountIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "account_ids不能为空")
		return
	}

	accountService := account.NewAccountService(repository.GetDB())
	err := accountService.Delete(userID, req.AccountIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "账号删除成功")
}

// CheckRequest 检测账号请求结构
type CheckRequest struct {
	AccountIDs []string `json:"account_ids" binding:"required,min=1"`
}

// Check 检测账号状态
func Check(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	var req CheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效:"+err.Error())
		return
	}

	// 添加参数验证
	if len(req.AccountIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "account_ids不能为空")
		return
	}

	accountService := account.NewAccountService(repository.GetDB())
	results, err := accountService.Check(c.Request.Context(), userID, req.AccountIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, results)
}

// ApplyHKRequest 申请HK区请求结构
type ApplyHKRequest struct {
	AccountIDs []string `json:"account_ids" binding:"required,min=1"`
}

// ApplyHK 申请开通香港区域
func ApplyHK(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	var req ApplyHKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	accountService := account.NewAccountService(repository.GetDB())
	results, err := accountService.ApplyHK(c.Request.Context(), userID, req.AccountIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, results)
}

// CreateInstanceRequest 创建实例请求结构
type CreateInstanceRequest struct {
	AccountIDs []string `json:"account_ids" binding:"required"`
	Region     string   `json:"region"` // 可选,默认ap-east-1
	Count      int32    `json:"count"`  // 可选,默认1
}

// CreateInstance 创建实例接口
func CreateInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误:"+err.Error())
		return
	}

	// 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 处理区域参数，支持中文和英文简写
	if req.Region != "" {
		switch req.Region {
		case "jp", "japan", "日本":
			req.Region = "ap-northeast-3" // 日本区域
		case "sg", "singapore", "新加坡":
			req.Region = "ap-southeast-1" // 新加坡区域
		case "hk", "hongkong", "香港":
			req.Region = "ap-east-1" // 香港区域
		}
	}

	// 如果未指定区域，默认使用账号的区域，服务层会处理这个逻辑

	// 设置默认实例数量
	if req.Count <= 0 {
		req.Count = 1 // 默认创建1个实例
	}

	// 调用服务
	svc := account.NewAccountService(repository.GetDB())
	results, err := svc.CreateInstance(c, userID, req.AccountIDs, req.Region, req.Count)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "创建实例失败:"+err.Error())
		return
	}

	response.Success(c, http.StatusOK, results)
}

// CleanMicroRequest 清理t2.micro和t3.micro实例请求结构
type CleanMicroRequest struct {
	AccountIDs []string `json:"account_ids" binding:"required,min=1"`
}

// CleanT3Micro 清理t2.micro和t3.micro实例
func CleanT3Micro(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	var req CleanMicroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效:"+err.Error())
		return
	}

	// 添加参数验证
	if len(req.AccountIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "account_ids不能为空")
		return
	}

	accountService := account.NewAccountService(repository.GetDB())
	results, err := accountService.CleanMicroInstances(c.Request.Context(), userID, req.AccountIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, results)
}
