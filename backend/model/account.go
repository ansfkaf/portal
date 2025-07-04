// model/account.go
package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Account AWS账号模型
type Account struct {
	ID         string     `gorm:"primarykey;type:varchar(255)" json:"id"`              // 自增ID
	UserID     string     `gorm:"type:varchar(255);not null" json:"user_id"`           // 用户ID
	Key1       string     `gorm:"type:varchar(255);not null" json:"key1"`              // Key1
	Key2       string     `gorm:"type:varchar(255);not null" json:"key2"`              // Key2
	Email      *string    `gorm:"type:varchar(255);default:null" json:"email"`         // 邮箱
	Password   *string    `gorm:"type:varchar(255);default:null" json:"password"`      // 密码
	Quatos     *string    `gorm:"type:varchar(255);default:null" json:"quatos"`        // 配额
	HK         *string    `gorm:"type:varchar(255);default:null" json:"hk"`            // HK区状态
	VMCount    *int       `gorm:"type:int;default:null" json:"vm_count"`               // 虚拟机数量
	Region     *string    `gorm:"type:varchar(255);default:'ap-east-1'" json:"region"` // 区域代码
	CreateTime *time.Time `gorm:"type:timestamp;default:null" json:"create_time"`      // 创建时间
}

// TableName 指定表名
func (Account) TableName() string {
	return "accounts"
}

// ImportResult 导入结果结构
type ImportResult struct {
	Summary struct {
		SuccessCount     int `json:"success_count"`      // 成功数量
		FailedCount      int `json:"failed_count"`       // 失败总数
		DuplicateCount   int `json:"duplicate_count"`    // 重复数量
		FormatErrorCount int `json:"format_error_count"` // 格式错误数量
	} `json:"summary"`
	Details struct {
		DuplicateList   []string `json:"duplicate_list,omitempty"`    // 重复账号列表
		FormatErrorList []string `json:"format_error_list,omitempty"` // 格式错误账号列表
	} `json:"details"`
}

// AccountInput 导入的账号信息
type AccountInput struct {
	Account  string
	Password string
	Key1     string
	Key2     string
	Region   string // 新增区域字段
}

// RegionMapping 区域名称到代码的映射
var RegionMapping = map[string]string{
	"香港":  "ap-east-1",
	"日本":  "ap-northeast-3",
	"新加坡": "ap-southeast-1",
}

// ParseAccountList 解析账号列表
func ParseAccountList(input string) ([]AccountInput, []string) {
	var accounts []AccountInput
	var errorLines []string

	// 按行分割
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		// 跳过空行
		if strings.TrimSpace(line) == "" {
			continue
		}

		account, err := parseAccountLine(line)
		if err != nil {
			errorLines = append(errorLines, err.Error())
			continue
		}
		accounts = append(accounts, account)
	}

	return accounts, errorLines
}

// parseAccountLine 修改错误提示格式
func parseAccountLine(line string) (AccountInput, error) {
	// 尝试用 ---- 分割
	parts := strings.Split(line, "----")
	if len(parts) != 4 && len(parts) != 5 {
		// 尝试用 --- 分割
		parts = strings.Split(line, "---")
		if len(parts) != 4 && len(parts) != 5 {
			return AccountInput{}, fmt.Errorf("%s", line) // 直接返回原始行
		}
	}

	// 清理每个字段的空白字符
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// 验证基本字段不能为空
	for i := 0; i < 4; i++ {
		if parts[i] == "" {
			return AccountInput{}, fmt.Errorf("%s", line) // 直接返回原始行
		}
	}

	// 创建账号信息实例并设置默认区域为香港
	accountInput := AccountInput{
		Account:  parts[0],
		Password: parts[1],
		Key1:     parts[2],
		Key2:     parts[3],
		Region:   "ap-east-1", // 默认为香港区域代码
	}

	// 如果存在第5个字段作为区域
	if len(parts) == 5 && parts[4] != "" {
		region := parts[4]

		// 处理简写格式
		switch strings.ToUpper(region) {
		case "HK":
			accountInput.Region = "ap-east-1" // 香港
		case "JP":
			accountInput.Region = "ap-northeast-3" // 日本
		case "SG":
			accountInput.Region = "ap-southeast-1" // 新加坡
		default:
			// 检查是否是完整的区域代码
			if region == "ap-east-1" || region == "ap-northeast-3" || region == "ap-southeast-1" {
				accountInput.Region = region
			} else if code, exists := RegionMapping[region]; exists {
				// 如果是中文区域名，转换为代码
				accountInput.Region = code
			}
		}
	}

	return accountInput, nil
}

// ValidateAndCreateAccounts 验证并创建账号
func ValidateAndCreateAccounts(db *gorm.DB, accounts []AccountInput, userID string) ImportResult {
	var result ImportResult
	now := time.Now()

	for _, input := range accounts {
		// 检查重复
		var count int64
		db.Model(&Account{}).Where("key1 = ? OR key2 = ?", input.Key1, input.Key2).Count(&count)
		if count > 0 {
			result.Summary.DuplicateCount++
			result.Summary.FailedCount++
			// 将完整的账号信息格式化为原始输入格式
			duplicateInfo := fmt.Sprintf("%s---%s---%s---%s", input.Account, input.Password, input.Key1, input.Key2)
			if input.Region != "ap-east-1" {
				duplicateInfo += fmt.Sprintf("---%s", input.Region)
			}
			result.Details.DuplicateList = append(result.Details.DuplicateList, duplicateInfo)
			continue
		}

		// 创建账号
		account := Account{
			UserID:     userID,
			Key1:       input.Key1,
			Key2:       input.Key2,
			CreateTime: &now,
		}
		if input.Account != "" {
			account.Email = &input.Account
		}
		if input.Password != "" {
			account.Password = &input.Password
		}
		// 设置区域信息
		account.Region = &input.Region

		if err := db.Create(&account).Error; err != nil {
			fmt.Printf("创建账号失败，账号: %s, 错误: %v\n", input.Account, err)
			result.Summary.FailedCount++
			continue
		}

		result.Summary.SuccessCount++
	}

	return result
}

// List 获取指定用户ID的所有账号列表
func List(db *gorm.DB, userID string) ([]Account, error) {
	var accounts []Account
	// 查询指定用户ID的所有账号,并按创建时间降序排序
	result := db.Where("user_id = ?", userID).Order("create_time DESC").Find(&accounts)
	if result.Error != nil {
		return nil, result.Error
	}
	return accounts, nil
}

// VerifyAccountOwnership 验证账号归属权
func VerifyAccountOwnership(db *gorm.DB, userID string, accountIDs []string) error {
	var count int64
	err := db.Model(&Account{}).
		Where("id IN ? AND user_id = ?", accountIDs, userID).
		Count(&count).Error
	if err != nil {
		return err
	}

	if int(count) != len(accountIDs) {
		return fmt.Errorf("存在无权操作的账号ID")
	}

	return nil
}

// DeleteAccounts 批量删除账号并验证所有权
func DeleteAccounts(db *gorm.DB, userID string, accountIDs []string) error {
	// 先验证所有权
	if err := VerifyAccountOwnership(db, userID, accountIDs); err != nil {
		return err
	}

	// 执行批量删除
	return db.Where("id IN ? AND user_id = ?", accountIDs, userID).Delete(&Account{}).Error
}

// GetAccountKeysByIDs 通过ID数组获取账号的key信息
func GetAccountKeysByIDs(db *gorm.DB, userID string, accountIDs []string) ([]Account, error) {
	var accounts []Account
	err := db.Select("id, key1, key2, region").Where("id IN ? AND user_id = ?", accountIDs, userID).Find(&accounts).Error
	return accounts, err
}

// UpdateAccountStatus 更新账号状态
func UpdateAccountStatus(db *gorm.DB, accountID string, quota, hkStatus string, instanceCount *int32) error {
	// 使用事务确保并发安全
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"quatos":   quota,
		"hk":       hkStatus,
		"vm_count": instanceCount,
	}

	if err := tx.Model(&Account{}).Where("id = ?", accountID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ListValidAccounts 获取指定用户的有效账号列表
func ListValidAccounts(db *gorm.DB, userID string) ([]Account, error) {
	var accounts []Account
	// 查询指定用户ID且配额状态不是"账号已失效"的所有账号
	result := db.Where("user_id = ? AND (quatos != '账号已失效' OR quatos IS NULL)", userID).
		Select("id, key1, key2, region"). // 只查询需要的字段，包括区域
		Find(&accounts)
	if result.Error != nil {
		return nil, result.Error
	}
	return accounts, nil
}

// BeforeCreate GORM 的钩子，在创建记录前自动设置 ID
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	// 查询当前最大 ID
	var maxID int
	if err := tx.Model(&Account{}).Select("COALESCE(MAX(CAST(id AS SIGNED)), 0)").Scan(&maxID).Error; err != nil {
		return err
	}

	// 设置新的 ID (最大ID + 1)
	a.ID = strconv.Itoa(maxID + 1)

	return nil
}

// GetRegionCode 根据区域名称获取区域代码
func GetRegionCode(regionName string) string {
	// 如果已经是区域代码，直接返回
	if regionName == "ap-east-1" || regionName == "ap-northeast-3" || regionName == "ap-southeast-1" {
		return regionName
	}

	// 如果是中文名称，转换为代码
	if code, exists := RegionMapping[regionName]; exists {
		return code
	}

	// 默认返回香港区域代码
	return "ap-east-1"
}
