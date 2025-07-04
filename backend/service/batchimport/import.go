// service/batchimport/import.go
package batchimport

import (
	"fmt"
	"portal/model"
	"portal/pkg/pool"
	"portal/repository/batchimport"

	"gorm.io/gorm"
)

type ImportService struct {
	repo *batchimport.ImportRepository
}

func NewImportService(db *gorm.DB) *ImportService {
	return &ImportService{
		repo: batchimport.NewImportRepository(db),
	}
}

// ImportAccounts 导入账号，添加 userID 参数
func (s *ImportService) ImportAccounts(content string, userID string) (*model.ImportResult, error) {
	// 解析账号列表
	accounts, errorLines := model.ParseAccountList(content)
	fmt.Printf("解析结果: 成功账号数=%d, 错误行数=%d\n", len(accounts), len(errorLines))

	var result model.ImportResult

	// 处理格式错误
	if len(errorLines) > 0 {
		result.Summary.FailedCount += len(errorLines)
		result.Summary.FormatErrorCount = len(errorLines)
		result.Details.FormatErrorList = errorLines
	}

	// 如果有正确格式的账号，执行导入
	if len(accounts) > 0 {
		fmt.Printf("开始导入 %d 个账号\n", len(accounts))
		importResult := s.repo.ImportAccounts(accounts, userID)
		result.Summary.SuccessCount = importResult.Summary.SuccessCount
		result.Summary.FailedCount += importResult.Summary.FailedCount
		result.Summary.DuplicateCount = importResult.Summary.DuplicateCount
		result.Details.DuplicateList = importResult.Details.DuplicateList

		// 只有成功导入账号时才更新账号池
		if importResult.Summary.SuccessCount > 0 {
			// 导入成功后刷新账号池，保留原有账号的错误备注
			s.refreshAccountPool()
		}
	}

	return &result, nil
}

// refreshAccountPool 刷新账号池，保留原有账号的错误备注
func (s *ImportService) refreshAccountPool() {
	// 获取账号池实例
	accountPool := pool.GetAccountPool()

	// 保存当前账号池中所有账号的错误标记信息
	errorNotes := make(map[string]struct {
		IsSkipped bool
		ErrorNote string
	})

	// 获取当前池中所有账号
	currentAccounts := accountPool.GetAllAccounts()
	for _, account := range currentAccounts {
		// 仅保存被标记为跳过的账号信息
		if account.IsSkipped {
			errorNotes[account.ID] = struct {
				IsSkipped bool
				ErrorNote string
			}{
				IsSkipped: account.IsSkipped,
				ErrorNote: account.ErrorNote,
			}
		}
	}

	// 从数据库重新加载账号到池中
	err := accountPool.LoadAccountsFromDB()
	if err != nil {
		fmt.Printf("刷新账号池时出错: %v\n", err)
		return
	}

	// 恢复原有账号的错误标记
	for id, info := range errorNotes {
		// 如果新加载的账号池中仍有此账号，则恢复其错误标记
		account := accountPool.GetAccount(id)
		if account != nil && info.IsSkipped {
			accountPool.MarkAccountFailed(id, info.ErrorNote)
		}
	}

	fmt.Printf("账号池已刷新，当前账号数: %d\n", accountPool.Size())
}
