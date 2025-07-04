// repository/setting/setting.go
package setting

import (
	"fmt"
	"portal/model"
	"sort"
	"strconv"

	"gorm.io/gorm"
)

type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{
		db: db,
	}
}

// GetSetting 获取设置
func (r *SettingRepository) GetSetting(userID string) (*model.Setting, error) {
	setting := &model.Setting{}
	err := r.db.Where("user_id = ?", userID).First(setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果找不到记录，创建一个新的默认设置
			setting = &model.Setting{
				UserID:       userID,
				Region:       "香港",           // 默认值
				InstanceType: "c5n.large",    // 默认值
				DiskSize:     20,             // 默认值
				Password:     "ASFsafs463@r", // 默认值
				Script:       "",             // 默认为空
				JpScript:     "",             // 日本区域脚本默认为空
				SgScript:     "",             // 新加坡区域脚本默认为空
			}
			if err := r.db.Create(setting).Error; err != nil {
				return nil, err
			}
			return setting, nil
		}
		return nil, err
	}
	return setting, nil
}

// UpdateSetting 更新设置
func (r *SettingRepository) UpdateSetting(userID string, req *model.UpdateSettingRequest) error {
	// 首先获取当前用户的设置
	setting := &model.Setting{}
	if err := r.db.Where("user_id = ?", userID).First(setting).Error; err != nil {
		// 如果找不到记录，创建新记录
		setting = &model.Setting{
			UserID: userID,
		}
	}

	// 创建更新map（包含所有需要更新的字段）
	updates := map[string]interface{}{
		"region":        req.Region,
		"instance_type": req.InstanceType,
		// "disk_size":     req.DiskSize,
		"password":  req.Password,
		"script":    req.Script,
		"jp_script": req.JpScript,
		"sg_script": req.SgScript,
	}

	// 更新或创建记录
	if err := r.db.Model(setting).Where("user_id = ?", userID).Updates(updates).Error; err != nil {
		return err
	}

	// 获取完整的更新后的设置
	if err := r.db.Where("user_id = ?", userID).First(setting).Error; err != nil {
		return err
	}

	// 验证密码强度
	if err := setting.ValidatePassword(); err != nil {
		return fmt.Errorf("密码验证失败: %v", err)
	}

	return nil
}

// GetAllSettings 获取所有用户的设置
func (r *SettingRepository) GetAllSettings() ([]*model.Setting, error) {
	var settings []*model.Setting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}

	// 对查询结果按照 user_id 进行排序（数字排序）
	sort.Slice(settings, func(i, j int) bool {
		// 尝试将 user_id 转换为整数进行比较
		idI, errI := strconv.Atoi(settings[i].UserID)
		idJ, errJ := strconv.Atoi(settings[j].UserID)

		// 如果两个 ID 都可以成功转换为整数，则按整数大小比较
		if errI == nil && errJ == nil {
			return idI < idJ
		}

		// 如果转换出错，则按字符串比较
		return settings[i].UserID < settings[j].UserID
	})

	return settings, nil
}
