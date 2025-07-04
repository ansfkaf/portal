// pkg/aws/client.go
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// AWSClient AWS客户端结构体
type AWSClient struct {
	AccessKey string
	SecretKey string
}

// NewAWSClient 创建一个新的AWS客户端
func NewAWSClient(accessKey, secretKey string) *AWSClient {
	return &AWSClient{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
}

// createConfig 创建特定区域的AWS配置
func (c *AWSClient) createConfig(ctx context.Context, region string) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.AccessKey,
			c.SecretKey,
			"",
		)),
	)
}
