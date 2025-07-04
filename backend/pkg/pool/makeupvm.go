// pkg/pool/makeupvm.go
package pool

import (
	"context"
	"fmt"
	"log"
	"portal/model"
	"portal/pkg/aws"
	"portal/repository"
	"strings"

	"gorm.io/gorm"
)

// InstanceCreationResult 创建实例的结果
type InstanceCreationResult struct {
	Success    bool   // 是否成功
	InstanceID string // 实例ID
	PublicIP   string // 公网IP
	Error      error  // 错误信息
}

// getAMIForRegion 根据区域获取对应的AMI ID
func getAMIForRegion(regionCode string) string {
	amiMap := map[string]string{
		"ap-east-1":      "ami-06dd48f3dbcc241f3", // 香港区域
		"ap-northeast-3": "ami-0eed40102c8eb6998", // 日本区域
		"ap-southeast-1": "ami-0acbb557db23991cc", // 新加坡区域
	}

	if ami, exists := amiMap[regionCode]; exists {
		return ami
	}

	// 默认返回香港区域的AMI
	return "ami-06dd48f3dbcc241f3"
}

// getScriptForRegion 根据区域获取对应的启动脚本
func getScriptForRegion(setting *model.Setting, regionCode string) string {
	// 根据区域选择对应的脚本
	switch regionCode {
	case "ap-northeast-3": // 日本区域
		return setting.JpScript // 直接返回日本区域脚本，无论是否为空
	case "ap-southeast-1": // 新加坡区域
		return setting.SgScript // 直接返回新加坡区域脚本，无论是否为空
	case "ap-east-1": // 香港区域
		return setting.Script // 香港区域使用通用脚本
	default:
		return "" // 其他未知区域返回空字符串
	}
}

// CreateInstanceForUser 为用户创建实例
// 增加 regionOverride 参数，允许指定区域覆盖用户设置
func CreateInstanceForUser(userID string, regionOverride string) (*InstanceCreationResult, error) {
	log.Printf("调试: 开始为用户[%s]创建实例，区域覆盖=[%s]", userID, regionOverride)

	// 获取数据库连接
	db := repository.GetDB()
	if db == nil {
		log.Printf("调试: 数据库连接为nil")
		return nil, fmt.Errorf("无法获取数据库连接")
	}

	// 获取账号池
	accountPool := GetAccountPool()
	if accountPool == nil {
		log.Printf("严重错误: 获取到的账号池为nil")
		return nil, fmt.Errorf("账号池无效")
	}

	log.Printf("调试: 创建实例前账号池状态: 总数=%d, 可用=%d",
		accountPool.Size(), accountPool.AvailableSize())

	// 获取用户设置
	setting, err := model.GetSettingByUserID(db, userID)
	if err != nil {
		log.Printf("调试: 获取用户设置失败: %v", err)
		return nil, fmt.Errorf("获取用户设置失败: %v", err)
	}

	// 获取区域代码，优先使用传入的区域覆盖
	regionCode := setting.GetRegionCode()
	if regionOverride != "" {
		regionCode = regionOverride
		log.Printf("用户[%s]使用指定区域[%s]进行补机，覆盖默认区域", userID, regionCode)
	}

	log.Printf("调试: 使用区域代码=[%s], 实例类型=[%s]", regionCode, setting.InstanceType)

	// 获取下一个可用账号，根据实例类型和区域选择适合的账号
	log.Printf("调试: 准备获取用户[%s]实例类型[%s]区域[%s]的账号",
		userID, setting.InstanceType, regionCode)
	account := accountPool.GetNextAccountForInstanceType(setting.InstanceType, regionCode)
	if account == nil {
		log.Printf("没有可用的账号，用户[%s]在区域[%s]的补机任务暂停，实例类型[%s]", userID, regionCode, setting.InstanceType)
		log.Printf("调试: 没有找到可用账号，账号池状态: 总数=%d, 可用=%d",
			accountPool.Size(), accountPool.AvailableSize())
		return nil, fmt.Errorf("没有可用的账号")
	}

	log.Printf("调试: 成功获取账号[%s]，准备创建AWS客户端", account.ID)

	// 创建AWS客户端
	awsClient := aws.NewAWSClient(account.Key1, account.Key2)
	// log.Printf("调试: AWS客户端已创建")

	// 获取区域对应的AMI
	amiID := getAMIForRegion(regionCode)
	log.Printf("调试: 区域[%s]的AMI ID=[%s]", regionCode, amiID)

	// 获取区域对应的脚本
	script := getScriptForRegion(setting, regionCode)
	// scriptLen := 0
	// if script != "" {
	// 	scriptLen = len(script)
	// }
	// log.Printf("调试: 获取到启动脚本，长度=%d字节", scriptLen)

	// 准备创建实例的参数
	params := aws.CreateInstanceParams{
		Region:       regionCode,              // 使用确定的区域代码
		ImageID:      amiID,                   // 根据区域获取对应的AMI
		InstanceType: setting.InstanceType,    // 从用户设置获取
		DiskSize:     int32(setting.DiskSize), // 从用户设置获取
		Password:     setting.Password,        // 从用户设置获取
		Count:        1,                       // 每次只开一台
		Script:       script,                  // 根据区域获取对应的脚本
		UserID:       userID,                  // 用于标签
		AccountID:    account.ID,              // 用于标签
	}
	// log.Printf("调试: 创建实例参数已准备完成")

	// 执行创建操作
	// log.Printf("调试: 准备调用AWS API创建实例")
	instances, err := awsClient.CreateInstance(context.Background(), params)
	if err != nil {
		errMsg := err.Error()
		log.Printf("使用账号[%s]在区域[%s]开机失败, 实例类型[%s]: %v", account.ID, regionCode, setting.InstanceType, err)
		log.Printf("调试: AWS创建实例失败: %v", err)

		// 处理错误
		log.Printf("调试: 处理账号错误，账号ID=%s", account.ID)
		handleAccountError(db, account.ID, errMsg, awsClient, setting.InstanceType, regionCode)
		log.Printf("调试: 账号错误处理完成")

		// 返回错误
		return nil, err
	}

	// 开机成功，更新账号的实例使用计数
	// log.Printf("调试: AWS创建实例成功，实例ID=%s", instances[0].InstanceID)
	// log.Printf("调试: 准备更新账号[%s]使用计数", account.ID)
	accountPool.IncrementInstanceUsage(account.ID, setting.InstanceType, regionCode)
	// log.Printf("调试: 账号使用计数已更新")

	// 开机成功
	log.Printf("用户[%s]使用账号[%s]在区域[%s]补机成功，实例类型[%s]，实例ID[%s]", userID, account.ID, regionCode, setting.InstanceType, instances[0].InstanceID)

	result := &InstanceCreationResult{
		Success:    true,
		InstanceID: instances[0].InstanceID,
		PublicIP:   instances[0].PublicIP,
	}

	log.Printf("调试: CreateInstanceForUser完成，成功=true，实例ID=%s", instances[0].InstanceID)
	return result, nil
}

// handleAccountError 处理账号错误
func handleAccountError(db *gorm.DB, accountID string, errMsg string, awsClient *aws.AWSClient, instanceType string, regionCode string) {
	accountPool := GetAccountPool()

	if strings.Contains(errMsg, "AuthFailure") ||
		strings.Contains(errMsg, "not able to validate the provided access credentials") {
		// 账号凭证无效，检查账号状态
		quota, checkErr := awsClient.GetEC2Quota(context.Background())

		if checkErr != nil || quota == "账号已失效" {
			log.Printf("账号[%s]已失效，更新状态并从账号池移除", accountID)
			// 更新数据库中的账号状态
			updateErr := model.UpdateAccountStatus(db, accountID, "账号已失效", "", nil)
			if updateErr != nil {
				log.Printf("更新账号[%s]状态失败: %v", accountID, updateErr)
			}

			// 从账号池中移除账号
			accountPool.RemoveAccount(accountID)
		} else {
			// 获取区域类型，只有香港区才需要检查区域是否开通
			if regionCode == "ap-east-1" { // 香港区
				// 账号有效但可能未开通香港区
				status, regionErr := awsClient.CheckRegionStatus(context.Background(), regionCode)

				if regionErr != nil || status != "启用" {
					// 尝试开通香港区
					enableErr := awsClient.EnableRegion(context.Background(), regionCode)
					if enableErr != nil {
						log.Printf("为账号[%s]开通香港区域失败: %v", accountID, enableErr)
					}

					// 标记账号需要跳过，稍后再重试
					accountPool.MarkAccountFailed(accountID, "香港区域未开通，已尝试开通")
				}
			} else {
				// 其他区域（日本、新加坡）不需要单独开通，但可能仍有其他凭证问题
				accountPool.MarkAccountFailed(accountID, fmt.Sprintf("%s区域凭证验证失败", regionCode))
			}
		}
	} else if strings.Contains(errMsg, "PendingVerification") {
		// 区域资源验证中
		accountPool.MarkAccountFailed(accountID, fmt.Sprintf("%s区域资源验证中", regionCode))
	} else if strings.Contains(errMsg, "VcpuLimitExceeded") ||
		strings.Contains(errMsg, "vCPU capacity") {
		// 配额用完 - 针对特定实例类型标记
		if instanceType == "c5n.xlarge" || instanceType == "c5n.2xlarge" || instanceType == "c5n.4xlarge" {
			// 大型实例配额不足，但可能小型实例仍可创建，只标记这个实例类型
			accountPool.MarkInstanceTypeFailed(accountID, instanceType, fmt.Sprintf("%s区域的%s配额已用完，跳过", regionCode, instanceType))
			log.Printf("账号[%s]在区域[%s]对实例类型[%s]配额不足，但可能仍可用于其他类型", accountID, regionCode, instanceType)
		} else {
			// 如果是小型实例也配额不足，整个账号标记为跳过
			accountPool.MarkAccountFailed(accountID, fmt.Sprintf("%s区域配额已用完，跳过", regionCode))
			log.Printf("账号[%s]在区域[%s]的所有实例类型配额均不足", accountID, regionCode)
		}
	} else {
		// 其他错误
		accountPool.MarkAccountFailed(accountID, fmt.Sprintf("%s区域开机失败: %s", regionCode, errMsg))
	}
}
