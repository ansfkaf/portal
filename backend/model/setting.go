// model/setting.go
package model

import (
	"errors"
	"regexp"

	"gorm.io/gorm"
)

// Setting 系统设置模型
type Setting struct {
	UserID       string `gorm:"primarykey;type:varchar(255)" json:"user_id"`                         // 用户ID作为主键
	Region       string `gorm:"type:varchar(255);not null;default:'香港'" json:"region"`               // 开机区域
	InstanceType string `gorm:"type:varchar(255);not null;default:'c5n.large'" json:"instance_type"` // 实例规格
	DiskSize     int    `gorm:"type:int;not null;default:20" json:"disk_size"`                       // 硬盘大小
	Password     string `gorm:"type:varchar(255);not null;default:'Aa33669900@@'" json:"password"`   // 开机密码
	Script       string `gorm:"type:text" json:"script"`                                             // 开机脚本
	JpScript     string `gorm:"type:text" json:"jp_script"`                                          // 日本区域开机脚本
	SgScript     string `gorm:"type:text" json:"sg_script"`                                          // 新加坡区域开机脚本
}

// UpdateSettingRequest 更新设置请求结构体
type UpdateSettingRequest struct {
	Region       string `json:"region"`
	InstanceType string `json:"instance_type"`
	DiskSize     int    `json:"disk_size"`
	Password     string `json:"password"`
	Script       string `json:"script"`
	JpScript     string `json:"jp_script"` // 日本区域开机脚本
	SgScript     string `json:"sg_script"` // 新加坡区域开机脚本
}

// TableName 指定表名
func (Setting) TableName() string {
	return "settings"
}

// GetRegionCode 获取区域对应的 AWS 区域代码
func (s *Setting) GetRegionCode() string {
	regionMap := map[string]string{
		"香港":  "ap-east-1",
		"日本":  "ap-northeast-3",
		"新加坡": "ap-southeast-1",
	}

	if code, exists := regionMap[s.Region]; exists {
		return code
	}
	return s.Region // 如果没有映射关系，返回原始值
}

// ValidatePassword 验证密码强度
func (s *Setting) ValidatePassword() error {
	if len(s.Password) < 6 {
		return errors.New("密码长度必须大于6位")
	}

	// 检查是否包含字母（不区分大小写）
	hasLetter, _ := regexp.MatchString(`[a-zA-Z]`, s.Password)
	if !hasLetter {
		return errors.New("密码必须包含至少一个字母")
	}

	return nil
}

// GetSettingByUserID 根据用户ID获取设置
func GetSettingByUserID(db *gorm.DB, userID string) (*Setting, error) {
	var setting Setting
	result := db.Where("user_id = ?", userID).First(&setting)
	if result.Error != nil {
		return nil, result.Error
	}
	return &setting, nil
}

// UpdateSettings 更新设置参数
func (s *Setting) UpdateSettings(db *gorm.DB) error {
	// 首先验证密码强度
	if err := s.ValidatePassword(); err != nil {
		return err
	}

	// 使用 user_id 更新记录
	result := db.Model(&Setting{}).Where("user_id = ?", s.UserID).Updates(map[string]interface{}{
		"region":        s.Region,
		"instance_type": s.InstanceType,
		"disk_size":     s.DiskSize,
		"password":      s.Password,
		"script":        s.Script,
		"jp_script":     s.JpScript,
		"sg_script":     s.SgScript,
	})

	if result.Error != nil {
		return result.Error
	}

	// 检查是否找到并更新了记录
	if result.RowsAffected == 0 {
		// 如果没有找到记录，创建新记录
		if err := db.Create(s).Error; err != nil {
			return err
		}
	}

	return nil
}
