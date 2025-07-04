// service/account/create_instance.go
package account

import (
	"context"
	"fmt"
	"portal/model"
	"portal/pkg/aws"
	"sync"
)

// CreateInstanceResult 创建实例结果
type CreateInstanceResult struct {
	AccountID string                     `json:"account_id"`
	Status    string                     `json:"status"`    // 成功/失败
	Message   string                     `json:"message"`   // 错误信息
	Instances []aws.CreateInstanceResult `json:"instances"` // 成功创建的实例信息
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
		if setting.JpScript != "" {
			return setting.JpScript
		}
	case "ap-southeast-1": // 新加坡区域
		if setting.SgScript != "" {
			return setting.SgScript
		}
	}

	// 默认使用通用脚本
	return setting.Script
}

// CreateInstance 批量创建实例
func (s *AccountService) CreateInstance(ctx context.Context, userID string, accountIDs []string, region string, count int32) ([]CreateInstanceResult, error) {
	// 验证账号归属权
	if err := model.VerifyAccountOwnership(s.repo.DB, userID, accountIDs); err != nil {
		return nil, err
	}

	// 获取账号的key信息和区域信息
	accounts, err := model.GetAccountKeysByIDs(s.repo.DB, userID, accountIDs)
	if err != nil {
		return nil, err
	}

	// 获取用户设置
	setting, err := model.GetSettingByUserID(s.repo.DB, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户设置失败: %v", err)
	}

	// 如果未指定数量，默认为1
	if count <= 0 {
		count = 1
	}

	var (
		results []CreateInstanceResult
		wg      sync.WaitGroup
		mu      sync.Mutex
	)

	// 对每个账号并发执行创建操作
	for _, acc := range accounts {
		wg.Add(1)
		go func(acc model.Account) {
			defer wg.Done()

			result := CreateInstanceResult{
				AccountID: acc.ID,
				Status:    "失败", // 默认状态为失败，只有成功执行才会改变
			}

			// 确定使用的区域代码
			var regionCode string
			if region != "" {
				// 如果请求指定了区域，使用请求的区域
				regionCode = region
			} else if acc.Region != nil && *acc.Region != "" {
				// 否则使用账号的区域
				regionCode = *acc.Region
			} else {
				// 如果账号没有区域，使用用户设置的区域代码
				regionCode = setting.GetRegionCode()
			}

			// 验证账号和区域匹配
			if acc.Region != nil && *acc.Region != "" && *acc.Region != regionCode {
				result.Message = fmt.Sprintf("账号[%s]区域为[%s]，与请求区域[%s]不匹配", acc.ID, *acc.Region, regionCode)

				// 线程安全地添加结果
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// 初始化AWS客户端
			awsClient := aws.NewAWSClient(acc.Key1, acc.Key2)

			// 获取区域对应的AMI
			amiID := getAMIForRegion(regionCode)

			// 获取区域对应的脚本
			script := getScriptForRegion(setting, regionCode)

			// 准备创建实例的参数
			params := aws.CreateInstanceParams{
				Region:       regionCode,              // 使用确定的区域代码
				ImageID:      amiID,                   // 根据区域获取对应的AMI
				InstanceType: setting.InstanceType,    // 从设置获取
				DiskSize:     int32(setting.DiskSize), // 从设置获取
				Password:     setting.Password,        // 从设置获取
				Count:        count,                   // 从请求参数获取
				Script:       script,                  // 根据区域获取对应的脚本
				UserID:       userID,                  // 用于标签
				AccountID:    acc.ID,                  // 用于标签
			}

			// 执行创建操作
			instances, err := awsClient.CreateInstance(ctx, params)
			if err != nil {
				result.Status = "失败"
				result.Message = err.Error()
			} else {
				result.Status = "成功"
				result.Instances = instances
			}

			// 线程安全地添加结果
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(acc)
	}

	// 等待所有创建操作完成
	wg.Wait()

	return results, nil
}
