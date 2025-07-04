// model/monitor.go
package model

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Monitor 监控配置模型
type Monitor struct {
	ID               uint   `gorm:"primaryKey;autoIncrement" json:"id"`                // 让数据库自动递增
	UserID           string `gorm:"type:varchar(255);not null" json:"user_id"`         // 用户ID
	Threshold        int    `gorm:"not null;default:0" json:"threshold"`               // 香港区阈值，默认为0
	JpThreshold      int    `gorm:"not null;default:0" json:"jp_threshold"`            // 日本区阈值，默认为0
	SgThreshold      int    `gorm:"not null;default:0" json:"sg_threshold"`            // 新加坡区阈值，默认为0
	IsEnabled        bool   `gorm:"not null;default:false" json:"is_enabled"`          // 监控开关，默认关闭
	IsTgEnabled      bool   `gorm:"not null;default:false" json:"is_tg_enabled"`       // TG通知开关，默认关闭
	TgUserID         string `gorm:"type:varchar(255);default:''" json:"tg_user_id"`    // TG用户ID，默认为空
	IsIPRangeEnabled bool   `gorm:"not null;default:false" json:"is_ip_range_enabled"` // IP段限制开关，默认关闭
	IPRange          string `gorm:"type:varchar(255);default:''" json:"ip_range"`      // 香港IP段，默认为空
	JpIPRange        string `gorm:"type:varchar(255);default:''" json:"jp_ip_range"`   // 日本IP段，默认为空
	SgIPRange        string `gorm:"type:varchar(255);default:''" json:"sg_ip_range"`   // 新加坡IP段，默认为空
}

// TableName 指定表名
func (Monitor) TableName() string {
	return "monitor"
}

// GetMonitorByUserID 获取用户的监控配置
func GetMonitorByUserID(db *gorm.DB, userID string) (*Monitor, error) {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果没有找到记录，创建默认配置
			config = Monitor{
				UserID:           userID,
				Threshold:        0,     // 默认香港区阈值为0
				JpThreshold:      0,     // 默认日本区阈值为0
				SgThreshold:      0,     // 默认新加坡区阈值为0
				IsEnabled:        true,  // 默认开启监控
				IsTgEnabled:      false, // 默认关闭TG通知
				TgUserID:         "",    // 默认TG用户ID为空
				IsIPRangeEnabled: false, // 默认关闭IP段限制
				IPRange:          "",    // 默认香港IP段为空
				JpIPRange:        "",    // 默认日本IP段为空
				SgIPRange:        "",    // 默认新加坡IP段为空
			}
			if err := db.Create(&config).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	}
	return &config, nil
}

// GetTgNotificationSettings 获取用户的TG通知设置
func GetTgNotificationSettings(db *gorm.DB, userID string) (bool, string, error) {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果没有找到记录，返回默认值
			return false, "", nil
		}
		return false, "", result.Error
	}
	return config.IsTgEnabled, config.TgUserID, nil
}

// GetIPRangeSettings 获取用户的IP段限制设置
func GetIPRangeSettings(db *gorm.DB, userID string) (bool, string, error) {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果没有找到记录，返回默认值
			return false, "", nil
		}
		return false, "", result.Error
	}
	return config.IsIPRangeEnabled, config.IPRange, nil
}

// GetIPRangeSettingsByRegion 根据区域获取用户的IP段限制设置
func GetIPRangeSettingsByRegion(db *gorm.DB, userID string, region string) (bool, string, error) {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果没有找到记录，返回默认值
			return false, "", nil
		}
		return false, "", result.Error
	}

	// 根据区域返回对应的IP范围
	var ipRange string
	switch region {
	case "ap-northeast-3": // 日本
		ipRange = config.JpIPRange
		// 如果日本区域没有设置IP范围，则使用香港区域的设置
		if ipRange == "" {
			ipRange = config.IPRange
		}
	case "ap-southeast-1": // 新加坡
		ipRange = config.SgIPRange
		// 如果新加坡区域没有设置IP范围，则使用香港区域的设置
		if ipRange == "" {
			ipRange = config.IPRange
		}
	default: // 默认香港区域
		ipRange = config.IPRange
	}

	return config.IsIPRangeEnabled, ipRange, nil
}

// GetAllMonitors 获取所有用户的监控配置
func GetAllMonitors(db *gorm.DB) ([]Monitor, error) {
	var configs []Monitor
	result := db.Find(&configs)
	if result.Error != nil {
		return nil, result.Error
	}
	return configs, nil
}

// UpdateMonitor 更新用户的监控配置（支持多区域阈值）
func UpdateMonitor(db *gorm.DB, userID string, threshold int, jpThreshold int, sgThreshold int, isEnabled bool) error {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，创建新记录
			config = Monitor{
				UserID:           userID,
				Threshold:        threshold,
				JpThreshold:      jpThreshold,
				SgThreshold:      sgThreshold,
				IsEnabled:        isEnabled,
				IsTgEnabled:      false, // 默认关闭TG通知
				TgUserID:         "",    // 默认TG用户ID为空
				IsIPRangeEnabled: false, // 默认关闭IP段限制
				IPRange:          "",    // 默认香港IP段为空
				JpIPRange:        "",    // 默认日本IP段为空
				SgIPRange:        "",    // 默认新加坡IP段为空
			}
			return db.Create(&config).Error
		}
		return result.Error
	}

	// 更新现有记录
	config.Threshold = threshold
	config.JpThreshold = jpThreshold
	config.SgThreshold = sgThreshold
	config.IsEnabled = isEnabled
	return db.Save(&config).Error
}

// 向后兼容的旧版更新函数
func UpdateMonitorLegacy(db *gorm.DB, userID string, threshold int, isEnabled bool) error {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，创建新记录
			config = Monitor{
				UserID:           userID,
				Threshold:        threshold,
				JpThreshold:      threshold, // 使用相同的阈值
				SgThreshold:      threshold, // 使用相同的阈值
				IsEnabled:        isEnabled,
				IsTgEnabled:      false, // 默认关闭TG通知
				TgUserID:         "",    // 默认TG用户ID为空
				IsIPRangeEnabled: false, // 默认关闭IP段限制
				IPRange:          "",    // 默认香港IP段为空
				JpIPRange:        "",    // 默认日本IP段为空
				SgIPRange:        "",    // 默认新加坡IP段为空
			}
			return db.Create(&config).Error
		}
		return result.Error
	}

	// 更新现有记录
	config.Threshold = threshold
	// 如果日本区和新加坡区阈值为0，也更新它们
	if config.JpThreshold == 0 {
		config.JpThreshold = threshold
	}
	if config.SgThreshold == 0 {
		config.SgThreshold = threshold
	}
	config.IsEnabled = isEnabled
	return db.Save(&config).Error
}

// UpdateTgSettings 更新用户的TG通知设置
func UpdateTgSettings(db *gorm.DB, userID string, isTgEnabled bool, tgUserID string) error {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，创建新记录
			config = Monitor{
				UserID:           userID,
				Threshold:        0,    // 默认阈值
				JpThreshold:      0,    // 默认日本区阈值
				SgThreshold:      0,    // 默认新加坡区阈值
				IsEnabled:        true, // 默认启用监控
				IsTgEnabled:      isTgEnabled,
				TgUserID:         tgUserID,
				IsIPRangeEnabled: false, // 默认关闭IP段限制
				IPRange:          "",    // 默认香港IP段为空
				JpIPRange:        "",    // 默认日本IP段为空
				SgIPRange:        "",    // 默认新加坡IP段为空
			}
			return db.Create(&config).Error
		}
		return result.Error
	}

	// 更新现有记录
	config.IsTgEnabled = isTgEnabled
	config.TgUserID = tgUserID
	return db.Save(&config).Error
}

// UpdateIPRangeSettings 更新用户的IP段限制设置
func UpdateIPRangeSettings(db *gorm.DB, userID string, isIPRangeEnabled bool, ipRange string) error {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，创建新记录
			config = Monitor{
				UserID:           userID,
				Threshold:        0,     // 默认阈值
				JpThreshold:      0,     // 默认日本区阈值
				SgThreshold:      0,     // 默认新加坡区阈值
				IsEnabled:        true,  // 默认启用监控
				IsTgEnabled:      false, // 默认关闭TG通知
				TgUserID:         "",    // 默认TG用户ID为空
				IsIPRangeEnabled: isIPRangeEnabled,
				IPRange:          ipRange, // 香港IP段
				JpIPRange:        "",      // 默认日本IP段为空
				SgIPRange:        "",      // 默认新加坡IP段为空
			}
			return db.Create(&config).Error
		}
		return result.Error
	}

	// 更新现有记录
	config.IsIPRangeEnabled = isIPRangeEnabled
	config.IPRange = ipRange
	return db.Save(&config).Error
}

// UpdateAllIPRangeSettings 更新用户的所有区域IP段限制设置
func UpdateAllIPRangeSettings(db *gorm.DB, userID string, isIPRangeEnabled bool, ipRange string, jpIPRange string, sgIPRange string) error {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，创建新记录
			config = Monitor{
				UserID:           userID,
				Threshold:        0,     // 默认阈值
				JpThreshold:      0,     // 默认日本区阈值
				SgThreshold:      0,     // 默认新加坡区阈值
				IsEnabled:        true,  // 默认启用监控
				IsTgEnabled:      false, // 默认关闭TG通知
				TgUserID:         "",    // 默认TG用户ID为空
				IsIPRangeEnabled: isIPRangeEnabled,
				IPRange:          ipRange,   // 香港IP段
				JpIPRange:        jpIPRange, // 日本IP段
				SgIPRange:        sgIPRange, // 新加坡IP段
			}
			return db.Create(&config).Error
		}
		return result.Error
	}

	// 更新现有记录
	config.IsIPRangeEnabled = isIPRangeEnabled
	config.IPRange = ipRange
	config.JpIPRange = jpIPRange
	config.SgIPRange = sgIPRange
	return db.Save(&config).Error
}

// UnbindTgUser 解绑用户的TG账号
func UnbindTgUser(db *gorm.DB, userID string) error {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，无需解绑
			return nil
		}
		return result.Error
	}

	// 清空TG用户ID并关闭TG通知
	config.TgUserID = ""
	config.IsTgEnabled = false
	return db.Save(&config).Error
}

// UpdateUserTgID 根据验证码更新用户的TG ID
func UpdateUserTgID(db *gorm.DB, userID string, tgUserID string) error {
	var config Monitor
	result := db.Where("user_id = ?", userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，创建新记录
			config = Monitor{
				UserID:           userID,
				Threshold:        0,    // 默认阈值
				JpThreshold:      0,    // 默认日本区阈值
				SgThreshold:      0,    // 默认新加坡区阈值
				IsEnabled:        true, // 默认启用监控
				IsTgEnabled:      true, // 绑定后默认开启TG通知
				TgUserID:         tgUserID,
				IsIPRangeEnabled: false, // 默认关闭IP段限制
				IPRange:          "",    // 默认香港IP段为空
				JpIPRange:        "",    // 默认日本IP段为空
				SgIPRange:        "",    // 默认新加坡IP段为空
			}
			return db.Create(&config).Error
		}
		return result.Error
	}

	// 更新现有记录
	config.TgUserID = tgUserID
	config.IsTgEnabled = true // 绑定后默认开启TG通知
	return db.Save(&config).Error
}

// BackupMonitorSettings 创建监控设置的临时备份表
func BackupMonitorSettings(db *gorm.DB) error {
	// 检查临时备份表是否已存在
	if db.Migrator().HasTable("monitor_backup") {
		// 如果已存在，先删除它
		if err := db.Migrator().DropTable("monitor_backup"); err != nil {
			return fmt.Errorf("删除已存在的备份表失败: %w", err)
		}
	}

	// 创建临时备份表 (与monitor表结构相同)
	if err := db.Table("monitor").Select("*").Table("monitor_backup").Create(nil).Error; err != nil {
		return fmt.Errorf("创建备份表结构失败: %w", err)
	}

	// 将数据从monitor表复制到monitor_backup表
	if err := db.Exec("INSERT INTO monitor_backup SELECT * FROM monitor").Error; err != nil {
		return fmt.Errorf("备份数据失败: %w", err)
	}

	// 关闭所有用户的TG通知
	if err := db.Exec("UPDATE monitor SET is_tg_enabled = false").Error; err != nil {
		return fmt.Errorf("关闭TG通知失败: %w", err)
	}

	return nil
}

// RestoreMonitorSettings 从临时备份表恢复监控设置
func RestoreMonitorSettings(db *gorm.DB) error {
	// 检查临时备份表是否存在
	if !db.Migrator().HasTable("monitor_backup") {
		return fmt.Errorf("备份表不存在，无法恢复")
	}

	// 从备份表恢复数据
	if err := db.Exec("UPDATE monitor m JOIN monitor_backup b ON m.user_id = b.user_id SET m.is_tg_enabled = b.is_tg_enabled, m.tg_user_id = b.tg_user_id").Error; err != nil {
		// 如果JOIN语法在当前数据库不支持，尝试使用标准SQL方式
		if innerErr := db.Exec("UPDATE monitor m SET m.is_tg_enabled = (SELECT b.is_tg_enabled FROM monitor_backup b WHERE m.user_id = b.user_id), m.tg_user_id = (SELECT b.tg_user_id FROM monitor_backup b WHERE m.user_id = b.user_id)").Error; innerErr != nil {
			return fmt.Errorf("恢复数据失败: %w", innerErr)
		}
	}

	// 删除临时备份表
	if err := db.Migrator().DropTable("monitor_backup"); err != nil {
		return fmt.Errorf("删除备份表失败: %w", err)
	}

	return nil
}

// GetThresholdByRegion 根据区域获取对应的阈值
func GetThresholdByRegion(config *Monitor, region string) int {
	switch region {
	case "ap-northeast-3": // 日本区域
		return config.JpThreshold
	case "ap-southeast-1": // 新加坡区域
		return config.SgThreshold
	default: // 默认香港区域或其他
		return config.Threshold
	}
}
