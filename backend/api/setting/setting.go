// api/setting/setting.go
package setting

import (
	"net/http"
	"portal/model"
	"portal/pkg/response"
	"portal/repository"
	"portal/service/setting"

	"github.com/gin-gonic/gin"
)

// AdminUpdateSettingRequest 管理员更新配置请求结构
type AdminUpdateSettingRequest struct {
	UserID       string `json:"user_id"`       // 要更新的用户ID
	Region       string `json:"region"`        // 区域
	InstanceType string `json:"instance_type"` // 实例类型
	DiskSize     int    `json:"disk_size"`     // 磁盘大小
	Password     string `json:"password"`      // 密码
	Script       string `json:"script"`        // 脚本
	JpScript     string `json:"jp_script"`     // 日本区域脚本
	SgScript     string `json:"sg_script"`     // 新加坡区域脚本
}

// GetSetting 获取设置
func GetSetting(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	settingService := setting.NewSettingService(repository.GetDB())
	// 使用用户ID获取设置
	result, err := settingService.GetSetting(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, result)
}

// UpdateSetting 更新设置
func UpdateSetting(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	var req model.UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// 预先验证密码强度
	tempSetting := &model.Setting{
		Password: req.Password,
	}
	if err := tempSetting.ValidatePassword(); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	settingService := setting.NewSettingService(repository.GetDB())
	err := settingService.UpdateSetting(userID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "设置更新成功")
}

// GetAllSettings 管理员接口：获取所有用户的设置
func GetAllSettings(c *gin.Context) {
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

	// 获取所有用户的设置
	settingService := setting.NewSettingService(repository.GetDB())
	settings, err := settingService.GetAllSettings()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取设置失败")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"total": len(settings),
		"list":  settings,
	})
}

// AdminUpdateSetting 管理员更新指定用户的设置
func AdminUpdateSetting(c *gin.Context) {
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
	var req AdminUpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证要更新的用户是否存在
	if req.UserID == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	// 预先验证密码强度
	tempSetting := &model.Setting{
		Password: req.Password,
	}
	if err := tempSetting.ValidatePassword(); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 转换为模型更新请求
	updateReq := &model.UpdateSettingRequest{
		Region:       req.Region,
		InstanceType: req.InstanceType,
		DiskSize:     req.DiskSize,
		Password:     req.Password,
		Script:       req.Script,
		JpScript:     req.JpScript,
		SgScript:     req.SgScript,
	}

	// 更新设置
	settingService := setting.NewSettingService(repository.GetDB())
	err := settingService.UpdateSetting(req.UserID, updateReq)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新设置失败: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "更新成功",
	})
}
