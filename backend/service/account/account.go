// service/account/account.go
package account

import (
	"context"
	"fmt"
	"portal/model"
	"portal/pkg/aws"
	"portal/pkg/pool"
	"portal/repository/account"
	"strings"
	"sync"

	"gorm.io/gorm"
)

type AccountService struct {
	repo *account.AccountRepository
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{
		repo: account.NewAccountRepository(db),
	}
}

// List 获取账号列表
func (s *AccountService) List(userID string) ([]model.Account, error) {
	return s.repo.List(userID)
}

// Delete 删除账号
func (s *AccountService) Delete(userID string, accountIDs []string) error {
	// 删除数据库中的账号
	err := s.repo.Delete(userID, accountIDs)

	// 如果删除成功，触发账号删除事件
	if err == nil {
		// 假设已导入 "portal/pkg/pool"
		for _, accountID := range accountIDs {
			pool.GetEventManager().TriggerEvent(pool.AccountDeleted, accountID)
		}
	}

	return err
}

type CheckResult struct {
	AccountID string `json:"account_id"`
	Quota     string `json:"quota"`
	HK        string `json:"hk"`       // 只有香港区账号才有值，表示HK区状态
	VMCount   *int32 `json:"vm_count"` // 虚拟机数量
	Region    string `json:"region"`   // 区域代码
}

func (s *AccountService) Check(ctx context.Context, userID string, accountIDs []string) ([]CheckResult, error) {
	// 验证账号归属权
	if err := model.VerifyAccountOwnership(s.repo.DB, userID, accountIDs); err != nil {
		return nil, err
	}

	// 获取账号的key信息
	accounts, err := model.GetAccountKeysByIDs(s.repo.DB, userID, accountIDs)
	if err != nil {
		return nil, err
	}

	// 创建并发控制
	var wg sync.WaitGroup
	// 设置合理的并发数量
	semaphore := make(chan struct{}, 10) // 最多10个并发
	resultChan := make(chan CheckResult, len(accounts))

	// 对每个账号并发执行检测
	for _, acc := range accounts {
		wg.Add(1)
		// 复制一份acc避免闭包问题
		account := acc

		go func() {
			defer wg.Done()

			// 获取信号量，控制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行单个账号检测
			result := s.checkSingleAccount(ctx, account)
			resultChan <- result
		}()
	}

	// 等待所有检测完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集所有结果
	var results []CheckResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results, nil
}

// checkSingleAccount 检测单个账号状态
func (s *AccountService) checkSingleAccount(ctx context.Context, acc model.Account) CheckResult {
	result := CheckResult{
		AccountID: acc.ID,
	}

	// 确保账号有区域信息
	regionCode := "ap-east-1" // 默认香港区域
	if acc.Region != nil && *acc.Region != "" {
		regionCode = *acc.Region
		result.Region = *acc.Region
	} else {
		result.Region = regionCode
	}

	// 初始化AWS客户端
	awsClient := aws.NewAWSClient(acc.Key1, acc.Key2)

	// 检查配额
	quota, err := awsClient.GetEC2Quota(ctx)
	fmt.Printf("账号ID: %s, 配额检测响应: %+v, 错误: %v\n", acc.ID, quota, err)
	if err != nil {
		// 判断凭证相关的错误
		if strings.Contains(err.Error(), "get credentials: failed") ||
			strings.Contains(err.Error(), "UnrecognizedClientException") {
			quota = "账号已失效"
		} else {
			quota = "查询失败"
		}
		// 更新数据库并返回结果
		model.UpdateAccountStatus(s.repo.DB, acc.ID, quota, "", nil)
		result.Quota = quota
		return result
	}
	result.Quota = quota

	// 如果账号已失效，不再检查其他状态
	if quota == "账号已失效" {
		model.UpdateAccountStatus(s.repo.DB, acc.ID, quota, "", nil)
		return result
	}

	// 只有香港区需要检查区域状态
	var hkStatus string
	if regionCode == "ap-east-1" {
		// 检查香港区域状态
		status, err := awsClient.CheckRegionStatus(ctx, regionCode)
		if err != nil {
			model.UpdateAccountStatus(s.repo.DB, acc.ID, quota, "查询失败", nil)
			result.HK = "查询失败"
			return result
		}
		hkStatus = status
		result.HK = status

		// 如果香港区未启用或启用中，不检查实例数
		if status == "未启用" || status == "启用中" {
			model.UpdateAccountStatus(s.repo.DB, acc.ID, quota, status, nil)
			return result
		}
	} else {
		// 日本和新加坡区域默认已启用，不需要单独检测
		hkStatus = "" // 非香港区账号，HK字段设为空
		result.HK = ""
	}

	// 检查实例数量
	count, err := awsClient.GetRunningInstanceCount(ctx, regionCode)
	if err != nil {
		model.UpdateAccountStatus(s.repo.DB, acc.ID, quota, hkStatus, nil)
		return result
	}
	result.VMCount = &count

	// 更新数据库
	model.UpdateAccountStatus(s.repo.DB, acc.ID, quota, hkStatus, &count)
	return result
}

// ApplyHKResult 申请HK区结果
type ApplyHKResult struct {
	AccountID string `json:"account_id"`
	Status    string `json:"status"`  // 操作状态：成功/失败
	Message   string `json:"message"` // 详细信息
}

// ApplyHK 批量申请开通HK区
func (s *AccountService) ApplyHK(ctx context.Context, userID string, accountIDs []string) ([]ApplyHKResult, error) {
	// 验证账号归属权
	if err := model.VerifyAccountOwnership(s.repo.DB, userID, accountIDs); err != nil {
		return nil, err
	}

	// 获取账号的key信息和区域信息
	accounts, err := model.GetAccountKeysByIDs(s.repo.DB, userID, accountIDs)
	if err != nil {
		return nil, err
	}

	var results []ApplyHKResult
	// 对每个账号执行申请操作
	for _, acc := range accounts {
		result := ApplyHKResult{
			AccountID: acc.ID,
		}

		// 获取区域代码
		regionCode := "ap-east-1" // 默认香港
		if acc.Region != nil && *acc.Region != "" {
			regionCode = *acc.Region
		}

		// 只有香港区域需要申请开通，其他区域直接跳过
		if regionCode != "ap-east-1" {
			result.Status = "成功"
			result.Message = "非香港区账号，无需申请开通"
			results = append(results, result)
			continue
		}

		// 初始化AWS客户端
		awsClient := aws.NewAWSClient(acc.Key1, acc.Key2)

		// 检查区域状态
		regionStatus, err := awsClient.CheckRegionStatus(ctx, regionCode)
		if err != nil {
			// 判断是否是账号失效
			if strings.Contains(err.Error(), "UnrecognizedClientException") ||
				strings.Contains(err.Error(), "InvalidClientTokenId") {
				result.Status = "失败"
				result.Message = "账号已失效"
				// 更新数据库状态
				model.UpdateAccountStatus(s.repo.DB, acc.ID, "账号已失效", "", nil)
			} else {
				result.Status = "失败"
				result.Message = "查询状态失败"
			}
			results = append(results, result)
			continue
		}

		// 根据状态执行不同操作
		switch regionStatus {
		case "未启用":
			// 调用开通操作
			err := awsClient.EnableRegion(ctx, regionCode)
			if err != nil {
				result.Status = "失败"
				result.Message = "开通失败: " + err.Error()
			} else {
				result.Status = "成功"
				result.Message = "已提交开通申请"
				// 更新数据库状态为启用中
				model.UpdateAccountStatus(s.repo.DB, acc.ID, "", "启用中", nil)
			}
		case "启用中":
			result.Status = "成功"
			result.Message = "正在启用中"
		case "启用":
			result.Status = "成功"
			result.Message = "香港区域已启用"
		default:
			result.Status = "失败"
			result.Message = "未知状态"
		}

		results = append(results, result)
	}

	return results, nil
}

// CleanMicroResult 清理微型实例结果
type CleanMicroResult struct {
	AccountID string `json:"account_id"`
	Status    string `json:"status"`     // 成功/失败
	Message   string `json:"message"`    // 详细信息
	Found     int    `json:"found"`      // 发现的实例数量
	Deleted   int    `json:"deleted"`    // 成功删除的实例数量
	T2Found   int    `json:"t2_found"`   // 发现的t2.micro实例数量
	T2Deleted int    `json:"t2_deleted"` // 成功删除的t2.micro实例数量
	T3Found   int    `json:"t3_found"`   // 发现的t3.micro实例数量
	T3Deleted int    `json:"t3_deleted"` // 成功删除的t3.micro实例数量
}

// CleanMicroSummary 清理微型实例汇总结果
type CleanMicroSummary struct {
	AccountResults []CleanMicroResult `json:"account_results"`  // 各账号处理结果
	TotalFound     int                `json:"total_found"`      // 总共发现的实例数量
	TotalDeleted   int                `json:"total_deleted"`    // 总共删除的实例数量
	T2TotalFound   int                `json:"t2_total_found"`   // 总共发现的t2.micro实例数量
	T2TotalDeleted int                `json:"t2_total_deleted"` // 总共删除的t2.micro实例数量
	T3TotalFound   int                `json:"t3_total_found"`   // 总共发现的t3.micro实例数量
	T3TotalDeleted int                `json:"t3_total_deleted"` // 总共删除的t3.micro实例数量
	SuccessCount   int                `json:"success_count"`    // 成功处理的账号数量
	FailCount      int                `json:"fail_count"`       // 失败的账号数量
}

// CleanMicroInstances 清理指定账号中的t2.micro和t3.micro实例
func (s *AccountService) CleanMicroInstances(ctx context.Context, userID string, accountIDs []string) (*CleanMicroSummary, error) {
	// 验证账号归属权
	if err := model.VerifyAccountOwnership(s.repo.DB, userID, accountIDs); err != nil {
		return nil, err
	}

	// 获取账号的key信息
	accounts, err := model.GetAccountKeysByIDs(s.repo.DB, userID, accountIDs)
	if err != nil {
		return nil, err
	}

	// 创建并发控制
	var wg sync.WaitGroup
	// 设置合理的并发数量
	semaphore := make(chan struct{}, 10) // 最多10个并发
	resultChan := make(chan CleanMicroResult, len(accounts))

	// 对每个账号并发执行检测和清理
	for _, acc := range accounts {
		wg.Add(1)
		// 复制一份acc避免闭包问题
		account := acc

		go func() {
			defer wg.Done()

			// 获取信号量，控制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行单个账号清理
			result := s.cleanMicroForAccount(ctx, account)
			resultChan <- result
		}()
	}

	// 等待所有处理完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集所有结果并汇总
	summary := &CleanMicroSummary{
		AccountResults: []CleanMicroResult{},
	}

	for result := range resultChan {
		summary.AccountResults = append(summary.AccountResults, result)

		// 更新汇总数据
		summary.TotalFound += result.Found
		summary.TotalDeleted += result.Deleted
		summary.T2TotalFound += result.T2Found
		summary.T2TotalDeleted += result.T2Deleted
		summary.T3TotalFound += result.T3Found
		summary.T3TotalDeleted += result.T3Deleted

		if result.Status == "成功" || result.Status == "部分成功" {
			summary.SuccessCount++
		} else {
			summary.FailCount++
		}
	}

	return summary, nil
}

// cleanMicroForAccount 清理单个账号内的t2.micro和t3.micro实例
func (s *AccountService) cleanMicroForAccount(ctx context.Context, acc model.Account) CleanMicroResult {
	result := CleanMicroResult{
		AccountID: acc.ID,
		Found:     0,
		Deleted:   0,
		T2Found:   0,
		T2Deleted: 0,
		T3Found:   0,
		T3Deleted: 0,
	}

	// 获取区域代码
	regionCode := "ap-east-1" // 默认香港
	if acc.Region != nil && *acc.Region != "" {
		regionCode = *acc.Region
	}

	// 初始化AWS客户端
	awsClient := aws.NewAWSClient(acc.Key1, acc.Key2)

	// 查询实例列表
	instances, err := awsClient.ListInstances(ctx, aws.ListInstancesParams{
		Region:    regionCode,
		AccountID: acc.ID,
	})

	if err != nil {
		result.Status = "失败"
		result.Message = "查询实例失败: " + err.Error()
		return result
	}

	// 筛选t2.micro和t3.micro实例
	var microInstances []aws.InstanceInfo
	for _, instance := range instances {
		if instance.InstanceType == "t2.micro" {
			microInstances = append(microInstances, instance)
			result.T2Found++
		} else if instance.InstanceType == "t3.micro" {
			microInstances = append(microInstances, instance)
			result.T3Found++
		}
	}

	result.Found = result.T2Found + result.T3Found

	// 如果没有找到微型实例，直接返回成功
	if result.Found == 0 {
		result.Status = "成功"
		result.Message = "未找到t2.micro或t3.micro实例"
		return result
	}

	// 删除找到的微型实例
	var deleteErrors []string
	for _, instance := range microInstances {
		// 执行删除操作
		params := aws.DeleteInstanceParams{
			Region:     regionCode,
			InstanceID: instance.InstanceID,
		}

		if err := awsClient.DeleteInstance(ctx, params); err != nil {
			deleteErrors = append(deleteErrors, fmt.Sprintf("实例%s删除失败: %s", instance.InstanceID, err.Error()))
		} else {
			result.Deleted++
			if instance.InstanceType == "t2.micro" {
				result.T2Deleted++
			} else if instance.InstanceType == "t3.micro" {
				result.T3Deleted++
			}
		}
	}

	// 根据删除结果设置状态
	if len(deleteErrors) == 0 {
		result.Status = "成功"
		if result.T2Found > 0 && result.T3Found > 0 {
			result.Message = fmt.Sprintf("成功清理%d个微型实例（t2.micro: %d, t3.micro: %d）",
				result.Deleted, result.T2Deleted, result.T3Deleted)
		} else if result.T2Found > 0 {
			result.Message = fmt.Sprintf("成功清理%d个t2.micro实例", result.T2Deleted)
		} else {
			result.Message = fmt.Sprintf("成功清理%d个t3.micro实例", result.T3Deleted)
		}
	} else {
		if result.Deleted > 0 {
			result.Status = "部分成功"
			result.Message = fmt.Sprintf("成功删除%d个实例（t2.micro: %d, t3.micro: %d），失败%d个，错误: %s",
				result.Deleted, result.T2Deleted, result.T3Deleted,
				result.Found-result.Deleted, strings.Join(deleteErrors, "; "))
		} else {
			result.Status = "失败"
			result.Message = fmt.Sprintf("所有%d个实例删除失败，错误: %s",
				result.Found, strings.Join(deleteErrors, "; "))
		}
	}

	return result
}

// 以下是CleanT3Micro相关代码，为了保持向后兼容，我们保留原来的函数
// 但是内部实现直接调用新的CleanMicroInstances函数

// CleanT3MicroResult 清理t3.micro实例结果
type CleanT3MicroResult struct {
	AccountID string `json:"account_id"`
	Status    string `json:"status"`  // 成功/失败
	Message   string `json:"message"` // 详细信息
	Found     int    `json:"found"`   // 发现的实例数量
	Deleted   int    `json:"deleted"` // 成功删除的实例数量
}

// CleanT3MicroSummary 清理t3.micro实例汇总结果
type CleanT3MicroSummary struct {
	AccountResults []CleanT3MicroResult `json:"account_results"` // 各账号处理结果
	TotalFound     int                  `json:"total_found"`     // 总共发现的实例数量
	TotalDeleted   int                  `json:"total_deleted"`   // 总共删除的实例数量
	SuccessCount   int                  `json:"success_count"`   // 成功处理的账号数量
	FailCount      int                  `json:"fail_count"`      // 失败的账号数量
}

// CleanT3Micro 清理指定账号中的t3.micro实例（为兼容旧版）
func (s *AccountService) CleanT3Micro(ctx context.Context, userID string, accountIDs []string) (*CleanT3MicroSummary, error) {
	// 调用新的实现
	newResult, err := s.CleanMicroInstances(ctx, userID, accountIDs)
	if err != nil {
		return nil, err
	}

	// 转换结果格式以兼容旧版接口
	oldResult := &CleanT3MicroSummary{
		AccountResults: make([]CleanT3MicroResult, len(newResult.AccountResults)),
		TotalFound:     newResult.TotalFound,
		TotalDeleted:   newResult.TotalDeleted,
		SuccessCount:   newResult.SuccessCount,
		FailCount:      newResult.FailCount,
	}

	for i, result := range newResult.AccountResults {
		oldResult.AccountResults[i] = CleanT3MicroResult{
			AccountID: result.AccountID,
			Status:    result.Status,
			Message:   result.Message,
			Found:     result.Found,
			Deleted:   result.Deleted,
		}
	}

	return oldResult, nil
}
