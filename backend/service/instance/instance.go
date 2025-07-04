// service/instance/instance.go
package instance

import (
	"context"
	"fmt"
	"portal/model"
	"portal/pkg/aws"
	"portal/repository/account"
	"sort"
	"strconv"
	"sync"

	"gorm.io/gorm"
)

type InstanceService struct {
	repo *account.AccountRepository
}

func NewInstanceService(db *gorm.DB) *InstanceService {
	return &InstanceService{
		repo: account.NewAccountRepository(db),
	}
}

// DeleteInstanceItem 删除实例项
type DeleteInstanceItem struct {
	AccountID  string
	Region     string // 区域参数可选
	InstanceID string
}

// DeleteResult 删除结果
type DeleteResult struct {
	AccountID  string `json:"account_id"`
	InstanceID string `json:"instance_id"`
	Status     string `json:"status"`  // 成功/失败
	Message    string `json:"message"` // 错误信息
}

// Delete 批量删除实例
func (s *InstanceService) Delete(ctx context.Context, userID string, instances []DeleteInstanceItem) ([]DeleteResult, error) {
	// 提取所有涉及的账号ID
	accountIDs := make([]string, 0)
	accountIDMap := make(map[string]bool)
	for _, instance := range instances {
		if !accountIDMap[instance.AccountID] {
			accountIDs = append(accountIDs, instance.AccountID)
			accountIDMap[instance.AccountID] = true
		}
	}

	// 不再验证账号归属权
	// 直接获取所有账号的key信息，不传入userID参数，避免筛选
	var accounts []model.Account
	var err error

	// 直接从数据库中获取账号信息，不检查用户ID，但需要包含区域信息
	err = s.repo.DB.Select("id, key1, key2, region").Where("id IN ?", accountIDs).Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	// 构建账号ID到账号信息的映射
	accountMap := make(map[string]model.Account)
	for _, acc := range accounts {
		accountMap[acc.ID] = acc
	}

	var (
		results []DeleteResult
		wg      sync.WaitGroup
		mu      sync.Mutex
	)

	// 并发删除实例
	for _, instance := range instances {
		wg.Add(1)
		go func(item DeleteInstanceItem) {
			defer wg.Done()

			result := DeleteResult{
				AccountID:  item.AccountID,
				InstanceID: item.InstanceID,
				Status:     "失败", // 默认状态为失败，只有成功执行才会改变
			}

			// 获取账号信息
			acc, exists := accountMap[item.AccountID]
			if !exists {
				result.Message = "账号不存在"
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// 确定使用的区域
			// 优先使用请求中指定的区域
			regionCode := item.Region

			// 如果请求中没有指定区域，则使用账号的区域
			if regionCode == "" && acc.Region != nil && *acc.Region != "" {
				regionCode = *acc.Region
			} else if regionCode == "" {
				// 如果账号也没有区域信息，使用香港作为默认值
				regionCode = "ap-east-1"
			}

			// 验证账号和区域是否匹配
			if acc.Region != nil && *acc.Region != "" && regionCode != *acc.Region {
				result.Message = fmt.Sprintf("账号[%s]区域为[%s]，与操作区域[%s]不匹配",
					acc.ID, *acc.Region, regionCode)

				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// 创建AWS客户端
			awsClient := aws.NewAWSClient(acc.Key1, acc.Key2)

			// 执行删除操作
			params := aws.DeleteInstanceParams{
				Region:     regionCode,
				InstanceID: item.InstanceID,
			}

			if err := awsClient.DeleteInstance(ctx, params); err != nil {
				result.Status = "失败"
				result.Message = err.Error()
			} else {
				result.Status = "成功"
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(instance)
	}

	wg.Wait()

	return results, nil
}

// ChangeIPItem 更换IP项
type ChangeIPItem struct {
	AccountID  string `json:"account_id"`
	Region     string `json:"region"` // 区域参数可选
	InstanceID string `json:"instance_id"`
}

// ChangeIPResult 更换IP结果
type ChangeIPResult struct {
	AccountID  string `json:"account_id"`
	InstanceID string `json:"instance_id"`
	Status     string `json:"status"`  // 成功/失败
	Message    string `json:"message"` // 错误信息
	OldIP      string `json:"old_ip"`  // 原IP
	NewIP      string `json:"new_ip"`  // 新IP
}

// ChangeIP 批量更换实例IP
func (s *InstanceService) ChangeIP(ctx context.Context, userID string, instances []ChangeIPItem) ([]ChangeIPResult, error) {
	// 提取所有涉及的账号ID
	accountIDs := make([]string, 0)
	accountIDMap := make(map[string]bool)
	for _, instance := range instances {
		if !accountIDMap[instance.AccountID] {
			accountIDs = append(accountIDs, instance.AccountID)
			accountIDMap[instance.AccountID] = true
		}
	}

	// 不再验证账号归属权
	// 直接获取所有账号的key信息，不传入userID参数，避免筛选
	var accounts []model.Account
	var err error

	// 直接从数据库中获取账号信息，不检查用户ID，但需要包含区域信息
	err = s.repo.DB.Select("id, key1, key2, region").Where("id IN ?", accountIDs).Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	// 构建账号ID到账号信息的映射
	accountMap := make(map[string]model.Account)
	for _, acc := range accounts {
		accountMap[acc.ID] = acc
	}

	var (
		results []ChangeIPResult
		wg      sync.WaitGroup
		mu      sync.Mutex
	)

	// 并发更换IP
	for _, instance := range instances {
		wg.Add(1)
		go func(item ChangeIPItem) {
			defer wg.Done()

			result := ChangeIPResult{
				AccountID:  item.AccountID,
				InstanceID: item.InstanceID,
				Status:     "失败", // 默认状态为失败，只有成功执行才会改变
			}

			// 获取账号信息
			acc, exists := accountMap[item.AccountID]
			if !exists {
				result.Message = "账号不存在"
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// 确定使用的区域
			// 优先使用请求中指定的区域
			regionCode := item.Region

			// 如果请求中没有指定区域，则使用账号的区域
			if regionCode == "" && acc.Region != nil && *acc.Region != "" {
				regionCode = *acc.Region
			} else if regionCode == "" {
				// 如果账号也没有区域信息，使用香港作为默认值
				regionCode = "ap-east-1"
			}

			// 验证账号和区域是否匹配
			if acc.Region != nil && *acc.Region != "" && regionCode != *acc.Region {
				result.Message = fmt.Sprintf("账号[%s]区域为[%s]，与操作区域[%s]不匹配",
					acc.ID, *acc.Region, regionCode)

				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// 创建AWS客户端
			awsClient := aws.NewAWSClient(acc.Key1, acc.Key2)

			// 执行更换IP操作
			params := aws.ChangeIPParams{
				Region:     regionCode,
				InstanceID: item.InstanceID,
			}

			changeResult, err := awsClient.ChangeIP(ctx, params)
			if err != nil {
				result.Status = "失败"
				result.Message = err.Error()
			} else {
				result.Status = "成功"
				result.OldIP = changeResult.OldIP
				result.NewIP = changeResult.NewIP
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(instance)
	}

	wg.Wait()

	return results, nil
}

// ListAccounts 获取用户的有效账号列表
func (s *InstanceService) ListAccounts(ctx context.Context, userID string) ([]model.Account, error) {
	accounts, err := model.ListValidAccounts(s.repo.DB, userID)
	if err != nil {
		return nil, fmt.Errorf("获取账号列表失败: %v", err)
	}

	// 按ID数值排序（先将字符串ID转为整数进行比较）
	sort.Slice(accounts, func(i, j int) bool {
		idI, errI := strconv.Atoi(accounts[i].ID)
		idJ, errJ := strconv.Atoi(accounts[j].ID)

		// 如果转换出错，则按字符串比较
		if errI != nil || errJ != nil {
			return accounts[i].ID < accounts[j].ID
		}

		return idI < idJ
	})

	return accounts, nil
}

// ListInstancesRequest 查询实例列表请求
type ListInstancesRequest struct {
	AccountIDs []string `json:"account_ids"`
	Region     string   `json:"region"` // 区域参数可选
}

// ListInstancesResult 查询实例列表结果
type ListInstancesResult struct {
	AccountID string             `json:"account_id"`
	Instances []aws.InstanceInfo `json:"instances"`
	Error     string             `json:"error,omitempty"`
}

// ListInstances 批量查询实例列表
func (s *InstanceService) ListInstances(ctx context.Context, userID string, req ListInstancesRequest) ([]ListInstancesResult, error) {
	// 验证账号归属权
	if err := model.VerifyAccountOwnership(s.repo.DB, userID, req.AccountIDs); err != nil {
		return nil, err
	}

	// 获取账号信息(包含区域)
	accounts, err := model.GetAccountKeysByIDs(s.repo.DB, userID, req.AccountIDs)
	if err != nil {
		return nil, err
	}

	var (
		results []ListInstancesResult
		wg      sync.WaitGroup
		mu      sync.Mutex
	)

	// 并发查询每个账号的实例
	for _, account := range accounts {
		wg.Add(1)
		go func(acc model.Account) {
			defer wg.Done()

			result := ListInstancesResult{
				AccountID: acc.ID,
			}

			// 确定查询使用的区域
			// 优先使用请求中指定的区域
			regionCode := req.Region

			// 如果请求中没有指定区域，则使用账号的区域
			if regionCode == "" && acc.Region != nil && *acc.Region != "" {
				regionCode = *acc.Region
			} else if regionCode == "" {
				// 如果账号也没有区域信息，使用香港作为默认值
				regionCode = "ap-east-1"
			}

			// 验证账号和区域是否匹配
			if acc.Region != nil && *acc.Region != "" && regionCode != *acc.Region {
				result.Error = fmt.Sprintf("账号[%s]区域为[%s]，与查询区域[%s]不匹配",
					acc.ID, *acc.Region, regionCode)

				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// 创建AWS客户端
			awsClient := aws.NewAWSClient(acc.Key1, acc.Key2)

			// 查询实例列表
			instances, err := awsClient.ListInstances(ctx, aws.ListInstancesParams{
				Region:    regionCode,
				AccountID: acc.ID,
			})

			if err != nil {
				result.Error = err.Error()
			} else {
				result.Instances = instances
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(account)
	}

	wg.Wait()
	return results, nil
}
