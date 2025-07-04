// api/batchimport/import.go
package batchimport

import (
	"net/http"
	"portal/pkg/response"
	"portal/repository"
	"portal/service/batchimport"

	"github.com/gin-gonic/gin"
)

// ImportRequest 导入请求结构
type ImportRequest struct {
	Content string `json:"content" binding:"required"` // 账号列表内容
}

// ImportAccounts 处理账号导入请求
func ImportAccounts(c *gin.Context) {
	// 从 context 获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "未获取到用户ID")
		return
	}

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求体格式无效")
		return
	}

	importService := batchimport.NewImportService(repository.GetDB())
	// 传入用户ID
	result, err := importService.ImportAccounts(req.Content, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, result)
}
