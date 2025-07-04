// api/instance/instance.go
package instance

import (
	"net/http"
	"portal/pkg/pool"
	"portal/pkg/response"
	"portal/repository"
	"portal/service/instance"

	"github.com/gin-gonic/gin"
)

// DeleteInstanceItem 单个实例的删除信息
type DeleteInstanceItem struct {
	AccountID  string `json:"account_id" binding:"required"`
	Region     string `json:"region"` // 移除 required 标签
	InstanceID string `json:"instance_id" binding:"required"`
}

// DeleteRequest 删除实例请求结构
type DeleteRequest struct {
	Instances []DeleteInstanceItem `json:"instances" binding:"required,min=1"`
}

// Delete 删除实例接口
func Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误:"+err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
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
	results, err := svc.Delete(c, userID, serviceInstances)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "删除实例失败:"+err.Error())
		return
	}

	response.Success(c, http.StatusOK, results)
}

// ChangeIPItem 单个实例的更换IP信息
type ChangeIPItem struct {
	AccountID  string `json:"account_id" binding:"required"`
	Region     string `json:"region"` // 移除 required 标签
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

	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
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
	results, err := svc.ChangeIP(c, userID, serviceInstances)
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

// ListAccounts 获取账号列表
func ListAccounts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	svc := instance.NewInstanceService(repository.GetDB())
	accounts, err := svc.ListAccounts(c, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取账号列表失败:"+err.Error())
		return
	}

	response.Success(c, http.StatusOK, accounts)
}

// ListInstances 实例列表查询接口
func ListInstances(c *gin.Context) {
	var req instance.ListInstancesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误:"+err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	// 如果提供了region参数，处理区域名称映射
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
	// 不再设置默认区域，让服务层根据账号信息决定使用哪个区域

	svc := instance.NewInstanceService(repository.GetDB())
	results, err := svc.ListInstances(c, userID, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询实例列表失败:"+err.Error())
		return
	}

	response.Success(c, http.StatusOK, results)
}
