// repository/account/account.go
package account

import (
	"portal/model"

	"gorm.io/gorm"
)

type AccountRepository struct {
	DB *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{
		DB: db,
	}
}

// List 获取账号列表
func (r *AccountRepository) List(userID string) ([]model.Account, error) {
	var accounts []model.Account
	err := r.DB.Where("user_id = ?", userID).Order("create_time DESC").Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// Delete 删除账号
func (r *AccountRepository) Delete(userID string, accountIDs []string) error {
	return model.DeleteAccounts(r.DB, userID, accountIDs)
}
