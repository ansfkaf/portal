// pkg/pool/iprange.go
package pool

import (
	"context"
	"log"
	"portal/model"
	"portal/service/instance"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// IPRangeChecker IP范围检测器结构体
type IPRangeChecker struct {
	db              *gorm.DB                  // 数据库连接
	instanceService *instance.InstanceService // 实例服务
	pool            *Pool                     // 连接池
	userMu          sync.Map                  // 用户级别的互斥锁映射
	userChecking    sync.Map                  // 用户当前是否正在检查的标记
}

// NewIPRangeChecker 创建新的IP范围检测器
func NewIPRangeChecker(db *gorm.DB, pool *Pool, instanceService *instance.InstanceService) *IPRangeChecker {
	return &IPRangeChecker{
		db:              db,
		instanceService: instanceService,
		pool:            pool,
		userMu:          sync.Map{},
		userChecking:    sync.Map{},
	}
}

// getUserLock 获取指定用户的互斥锁
func (c *IPRangeChecker) getUserLock(userID string) *sync.Mutex {
	actual, _ := c.userMu.LoadOrStore(userID, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// isUserBeingChecked 检查用户是否正在被检查
func (c *IPRangeChecker) isUserBeingChecked(userID string) bool {
	if value, ok := c.userChecking.Load(userID); ok {
		return value.(bool)
	}
	return false
}

// setUserCheckingStatus 设置用户的检查状态
func (c *IPRangeChecker) setUserCheckingStatus(userID string, checking bool) {
	c.userChecking.Store(userID, checking)
}

// isIPMatchRange 检查IP是否符合指定的IP范围前缀
func (c *IPRangeChecker) isIPMatchRange(ip string, ipRange string) bool {
	// 如果IP范围为空，则所有IP都符合
	if ipRange == "" {
		return true
	}

	// 检查IP是否以指定范围开头
	return strings.HasPrefix(ip, ipRange)
}

// CheckSingleUser 检查单个用户的实例IP范围
func (c *IPRangeChecker) CheckSingleUser(ctx context.Context, userID string) error {
	// 检查用户是否已经在检查中
	if c.isUserBeingChecked(userID) {
		return nil
	}

	// 获取用户锁并加锁
	userLock := c.getUserLock(userID)
	userLock.Lock()
	defer userLock.Unlock()

	// 设置用户正在检查的状态
	c.setUserCheckingStatus(userID, true)
	defer c.setUserCheckingStatus(userID, false)

	// 1. 检查用户的IP范围设置
	config, err := model.GetMonitorByUserID(c.db, userID)
	if err != nil {
		log.Printf("获取用户[%s]的监控配置失败: %v", userID, err)
		return err
	}

	// 如果IP范围限制未启用，则跳过
	if !config.IsIPRangeEnabled {
		return nil
	}

	// 2. 获取用户当前在线实例 - 只在开始时读取一次
	instances := c.pool.GetInstancesByUserID(userID)
	if len(instances) == 0 {
		return nil
	}

	// log.Printf("检查用户[%s]的IP范围限制: 共%d台实例", userID, len(instances))

	// 3. 逐个检查实例IP
	for _, inst := range instances { // 注意这里用 inst 避免与包名冲突
		// 获取当前实例所在区域对应的IP范围
		var ipRange string
		switch inst.Region {
		case "ap-northeast-3": // 日本区域
			ipRange = config.JpIPRange
		case "ap-southeast-1": // 新加坡区域
			ipRange = config.SgIPRange
		default: // 默认香港区域
			ipRange = config.IPRange
		}

		// 如果当前区域没有设置IP范围，则跳过该实例
		if ipRange == "" {
			continue
		}

		// 检查IP是否符合范围
		if c.isIPMatchRange(inst.IPv4, ipRange) {
			continue
		}

		log.Printf("实例[%s]的IP不符合范围要求，准备更换IP", inst.InstanceID)

		// 尝试更换IP，最多120次
		for i := 0; i < 120; i++ {
			// 直接构造更换IP的请求项
			changeIPItems := []instance.ChangeIPItem{
				{
					AccountID:  inst.AccountID,
					Region:     inst.Region,
					InstanceID: inst.InstanceID,
				},
			}

			results, err := c.instanceService.ChangeIP(ctx, inst.UserID, changeIPItems)
			if err != nil {
				log.Printf("更换实例[%s]IP失败: %v", inst.InstanceID, err)
				break
			}

			// 检查返回结果
			if len(results) == 0 {
				log.Printf("更换实例[%s]IP没有返回结果", inst.InstanceID)
				break
			}

			result := results[0]

			// 从 result 中提取状态和新IP
			if result.Status != "成功" {
				log.Printf("更换实例[%s]IP操作失败: %v", inst.InstanceID, result.Message)
				break
			}

			newIP := result.NewIP
			if newIP == "" {
				log.Printf("实例[%s]无法获取新IP", inst.InstanceID)
				break
			}

			// 检查新IP是否符合范围
			if c.isIPMatchRange(newIP, ipRange) {
				log.Printf("成功将实例[%s]的IP更换为符合要求的IP: %s", inst.InstanceID, newIP)
				break
			}

			log.Printf("实例[%s]新IP[%s]不符合要求[%s]，等待60秒后重试...", inst.InstanceID, newIP, ipRange)
			time.Sleep(60 * time.Second)
		}
	}

	return nil
}

// CheckAllUsers 检查所有用户的实例IP范围
func (c *IPRangeChecker) CheckAllUsers(ctx context.Context) error {
	// 1. 获取所有启用了IP范围限制的用户
	monitors, err := model.GetAllMonitors(c.db)
	if err != nil {
		log.Printf("获取监控配置失败: %v", err)
		return err
	}

	var enabledUsers []string
	for _, monitor := range monitors {
		if monitor.IsIPRangeEnabled {
			// 检查用户是否已经在检查中
			if !c.isUserBeingChecked(monitor.UserID) {
				enabledUsers = append(enabledUsers, monitor.UserID)
			}
		}
	}

	if len(enabledUsers) == 0 {
		return nil
	}

	// log.Printf("开始IP范围检查，共%d个用户", len(enabledUsers))

	// 2. 并发检查每个用户
	var wg sync.WaitGroup
	for _, userID := range enabledUsers {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()
			if err := c.CheckSingleUser(ctx, uid); err != nil {
				log.Printf("检查用户[%s]失败: %v", uid, err)
			}
		}(userID)
	}

	wg.Wait()
	// log.Printf("IP范围检查完成")
	return nil
}
