// service/setting/setting.go
package setting

import (
	"fmt"
	"portal/model"
	"portal/repository/setting"

	"gorm.io/gorm"
)

type SettingService struct {
	repo *setting.SettingRepository
}

func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{
		repo: setting.NewSettingRepository(db),
	}
}

// GetSetting 获取设置
func (s *SettingService) GetSetting(userID string) (interface{}, error) {
	return s.repo.GetSetting(userID)
}

// UpdateSetting 更新设置
func (s *SettingService) UpdateSetting(userID string, req interface{}) error {
	updateReq, ok := req.(*model.UpdateSettingRequest)
	if !ok {
		return fmt.Errorf("服务层数据转换失败")
	}

	return s.repo.UpdateSetting(userID, updateReq)
}

// GetAllSettings 获取所有用户的设置
func (s *SettingService) GetAllSettings() ([]*model.Setting, error) {
	return s.repo.GetAllSettings()
}
