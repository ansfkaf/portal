// pkg/aws/account.go
package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
)

// 区域常量
const (
	RegionHK    = "ap-east-1"      // 香港区域
	RegionJP    = "ap-northeast-3" // 日本区域
	RegionSG    = "ap-southeast-1" // 新加坡区域
	RegionQuota = "us-east-1"      // 配额查询区域
)

// GetEC2Quota 查询账号在指定区域的EC2配额
func (c *AWSClient) GetEC2Quota(ctx context.Context) (string, error) {
	// 创建美区配置用于查询配额
	cfg, err := c.createConfig(ctx, RegionQuota)
	if err != nil {
		return "", fmt.Errorf("加载AWS配置失败: %v", err)
	}

	quotaClient := servicequotas.NewFromConfig(cfg)

	input := &servicequotas.GetServiceQuotaInput{
		ServiceCode: aws.String("ec2"),
		QuotaCode:   aws.String("L-1216C47A"),
	}

	result, err := quotaClient.GetServiceQuota(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "UnrecognizedClientException") ||
			strings.Contains(err.Error(), "InvalidClientTokenId") {
			return "账号已失效", nil
		}
		return "", fmt.Errorf("查询配额失败: %v", err)
	}

	if result.Quota == nil || result.Quota.Value == nil {
		return "", fmt.Errorf("未找到配额信息")
	}

	return fmt.Sprintf("%d", int(*result.Quota.Value)), nil
}

// CheckRegionStatus 检查指定区域状态
func (c *AWSClient) CheckRegionStatus(ctx context.Context, regionCode string) (string, error) {
	// 创建指定区域配置
	cfg, err := c.createConfig(ctx, regionCode)
	if err != nil {
		return "", fmt.Errorf("加载AWS配置失败: %v", err)
	}

	accountClient := account.NewFromConfig(cfg)

	input := &account.GetRegionOptStatusInput{
		RegionName: aws.String(regionCode),
	}

	result, err := accountClient.GetRegionOptStatus(ctx, input)
	if err != nil {
		return "", fmt.Errorf("查询区域状态失败: %v", err)
	}

	switch result.RegionOptStatus {
	case "ENABLED":
		return "启用", nil
	case "ENABLING":
		return "启用中", nil
	case "DISABLED":
		return "未启用", nil
	default:
		return "未知状态", nil
	}
}

// GetRunningInstanceCount 获取指定区域运行中的实例数量
func (c *AWSClient) GetRunningInstanceCount(ctx context.Context, regionCode string) (int32, error) {
	// 创建指定区域配置
	cfg, err := c.createConfig(ctx, regionCode)
	if err != nil {
		return 0, fmt.Errorf("加载AWS配置失败: %v", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []string{
					"pending",
					"running",
					"stopping",
					"stopped",
				},
			},
		},
	}

	var count int32
	paginator := ec2.NewDescribeInstancesPaginator(ec2Client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("获取实例列表失败: %v", err)
		}

		for _, reservation := range output.Reservations {
			count += int32(len(reservation.Instances))
		}
	}

	return count, nil
}

// EnableRegion 开通指定区域
func (c *AWSClient) EnableRegion(ctx context.Context, regionCode string) error {
	// 创建指定区域配置
	cfg, err := c.createConfig(ctx, regionCode)
	if err != nil {
		return fmt.Errorf("加载AWS配置失败: %v", err)
	}

	accountClient := account.NewFromConfig(cfg)

	input := &account.EnableRegionInput{
		RegionName: aws.String(regionCode),
	}

	_, err = accountClient.EnableRegion(ctx, input)
	if err != nil {
		return fmt.Errorf("开通区域 %s 失败: %v", regionCode, err)
	}

	return nil
}
