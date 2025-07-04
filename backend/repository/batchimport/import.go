// repository/batchimport/import.go
package batchimport

import (
	"portal/model"

	"gorm.io/gorm"
)

type ImportRepository struct {
	db *gorm.DB
}

func NewImportRepository(db *gorm.DB) *ImportRepository {
	return &ImportRepository{
		db: db,
	}
}

// ImportAccounts 执行账号导入
func (r *ImportRepository) ImportAccounts(accounts []model.AccountInput, userID string) model.ImportResult {
	return model.ValidateAndCreateAccounts(r.db, accounts, userID)
}
