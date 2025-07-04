// model/user.go
package model

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID       string `gorm:"primarykey;type:varchar(255)" json:"id"`             // 自增ID
	Email    string `gorm:"type:varchar(255);unique;not null" json:"email"`     // 邮箱
	Password string `gorm:"type:varchar(255);not null" json:"-"`                // 密码（加密存储）
	IsAdmin  uint8  `gorm:"type:tinyint(1);default:0;not null" json:"is_admin"` // 是否是管理员（1是，0不是）
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate GORM hook，在创建记录前处理ID、密码和邮箱
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 查询当前最大 ID
	var maxID int
	if err := tx.Model(&User{}).Select("COALESCE(MAX(CAST(id AS SIGNED)), 0)").Scan(&maxID).Error; err != nil {
		return err
	}

	// 设置新的 ID (确保不是空字符串)
	newID := maxID + 1
	u.ID = strconv.Itoa(newID)

	// 验证邮箱格式
	if err := u.ValidateEmail(); err != nil {
		return err
	}
	// 验证密码强度
	if err := u.ValidatePassword(); err != nil {
		return err
	}
	// 对密码进行哈希处理
	return u.HashPassword()
}

// AfterCreate GORM hook，在用户创建成功后初始化相关配置
func (u *User) AfterCreate(tx *gorm.DB) error {
	// 使用事务确保数据一致性
	return tx.Transaction(func(tx *gorm.DB) error {
		// 初始化Setting表
		if err := u.initUserSettings(tx); err != nil {
			return err
		}

		// 初始化Monitor表
		if err := u.initUserMonitor(tx); err != nil {
			return err
		}

		return nil
	})
}

// 初始化用户Setting表
func (u *User) initUserSettings(tx *gorm.DB) error {
	setting := Setting{
		UserID:       u.ID,
		Region:       "香港",
		InstanceType: "c5n.large",
		DiskSize:     20,
		Password:     "ASFsafs463@r",
		Script:       "",
	}

	return tx.Create(&setting).Error
}

// 初始化用户Monitor表
func (u *User) initUserMonitor(tx *gorm.DB) error {
	monitor := Monitor{
		UserID:      u.ID,
		Threshold:   0,
		IsEnabled:   false,
		IsTgEnabled: false,
		TgUserID:    "",
	}
	return tx.Create(&monitor).Error // 让数据库自动生成 ID
}

// BeforeUpdate GORM hook，在更新记录前处理密码和邮箱
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// 如果邮箱被修改，验证新邮箱
	if tx.Statement.Changed("Email") {
		if err := u.ValidateEmail(); err != nil {
			return err
		}
	}
	// 如果密码字段被修改，则重新哈希
	if tx.Statement.Changed("Password") {
		// 验证密码强度
		if err := u.ValidatePassword(); err != nil {
			return err
		}
		// 对密码进行哈希处理
		return u.HashPassword()
	}
	return nil
}

// ValidateEmail 验证邮箱格式
func (u *User) ValidateEmail() error {
	if len(u.Email) == 0 {
		return errors.New("邮箱不能为空")
	}

	// 将邮箱转换为小写
	u.Email = strings.ToLower(u.Email)

	// 邮箱格式验证正则表达式
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	if !emailRegex.MatchString(u.Email) {
		return errors.New("邮箱格式不正确")
	}

	return nil
}

// HashPassword 对密码进行哈希处理
func (u *User) HashPassword() error {
	if len(u.Password) == 0 {
		return errors.New("密码不能为空")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码是否正确
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// PasswordConfig 密码配置
type PasswordConfig struct {
	MinLength      int  // 最小长度
	RequireNumber  bool // 要求数字
	RequireLetter  bool // 要求字母
	RequireSpecial bool // 要求特殊字符
}

// ValidatePasswordWithConfig 使用自定义配置验证密码强度
func (u *User) ValidatePasswordWithConfig(config PasswordConfig) error {
	if len(u.Password) < config.MinLength {
		return fmt.Errorf("密码长度必须至少为%d个字符", config.MinLength)
	}

	if config.RequireNumber {
		hasNumber := regexp.MustCompile(`[0-9]`).MatchString(u.Password)
		if !hasNumber {
			return errors.New("密码必须包含至少一个数字")
		}
	}

	if config.RequireLetter {
		hasLetter := regexp.MustCompile(`[A-Za-z]`).MatchString(u.Password)
		if !hasLetter {
			return errors.New("密码必须包含至少一个字母")
		}
	}

	if config.RequireSpecial {
		hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(u.Password)
		if !hasSpecial {
			return errors.New("密码必须包含至少一个特殊字符")
		}
	}

	return nil
}

// ValidatePassword 验证密码强度（使用默认配置）
func (u *User) ValidatePassword() error {
	// 使用默认配置
	config := PasswordConfig{
		MinLength:      6,
		RequireNumber:  true,
		RequireLetter:  true,
		RequireSpecial: false,
	}

	// 调试信息
	fmt.Printf("ValidatePassword 检查的密码: '%s'，长度: %d\n", u.Password, len(u.Password))

	return u.ValidatePasswordWithConfig(config)
}

// GetUsersByIDs 根据用户ID列表获取用户信息
func GetUsersByIDs(db *gorm.DB, ids []string) ([]map[string]interface{}, error) {
	var users []User
	result := db.Where("id IN ?", ids).Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}

	// 转换为简化的用户信息列表
	userInfos := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		userInfos = append(userInfos, map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		})
	}

	return userInfos, nil
}

// GetAllUsers 获取所有用户信息
func GetAllUsers(db *gorm.DB) ([]map[string]interface{}, error) {
	var users []User
	result := db.Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}

	// 转换为简化的用户信息列表
	userInfos := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		userInfos = append(userInfos, map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		})
	}

	return userInfos, nil
}

// UpdateUsers 更新用户信息（密码、邮箱、管理员状态）
func UpdateUsers(db *gorm.DB, userUpdates map[string]interface{}) error {
	// 获取用户IDs
	userIDs, ok := userUpdates["ids"].([]string)
	if !ok || len(userIDs) == 0 {
		return errors.New("未提供有效的用户ID列表")
	}

	// 创建更新映射
	updates := make(map[string]interface{})
	needDirectUpdate := false

	// 处理密码更新
	if password, exists := userUpdates["password"].(string); exists && password != "" {
		// 创建临时用户对象验证密码强度
		tempUser := &User{Password: password}

		// 验证密码
		if err := tempUser.ValidatePassword(); err != nil {
			return err
		}

		// 对密码进行哈希处理
		if err := tempUser.HashPassword(); err != nil {
			return err
		}

		// 使用哈希后的密码
		updates["password"] = tempUser.Password
		needDirectUpdate = true
	}

	// 处理邮箱更新
	if email, exists := userUpdates["email"].(string); exists && email != "" {
		// 创建临时用户对象验证邮箱格式
		tempUser := &User{Email: email}
		if err := tempUser.ValidateEmail(); err != nil {
			return err
		}

		updates["email"] = tempUser.Email
	}

	// 处理管理员状态更新
	if isAdmin, exists := userUpdates["is_admin"].(uint8); exists {
		updates["is_admin"] = isAdmin
	}

	// 检查是否有需要更新的字段
	if len(updates) == 0 {
		return errors.New("没有提供要更新的字段")
	}

	// 如果包含密码字段，直接执行SQL更新避免触发BeforeUpdate钩子
	if needDirectUpdate {
		// 开始事务
		return db.Transaction(func(tx *gorm.DB) error {
			for k, v := range updates {
				// 为每个字段构建SQL
				sql := fmt.Sprintf("UPDATE users SET %s = ? WHERE id IN (?)", k)
				if err := tx.Exec(sql, v, userIDs).Error; err != nil {
					return err
				}
			}
			return nil
		})
	}

	// 没有密码更新时，使用正常的Updates方法
	return db.Model(&User{}).Where("id IN ?", userIDs).Updates(updates).Error
}

// CreateUser 创建新用户
func CreateUser(db *gorm.DB, email string, password string, isAdmin uint8) (*User, error) {
	// 创建用户实例
	user := &User{
		Email:    email,
		Password: password,
		IsAdmin:  isAdmin,
	}

	// 保存到数据库，会触发 BeforeCreate 和 AfterCreate 钩子
	result := db.Create(user)
	if result.Error != nil {
		return nil, result.Error
	}

	return user, nil
}
