// pkg/pool/pool.go
package pool

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"portal/pkg/tg"
	"portal/repository"

	"github.com/gorilla/websocket"
)

// InstanceMetadata 实例元数据结构，与客户端上报的数据结构保持一致
type InstanceMetadata struct {
	InstanceID   string    `json:"instance_id"`   // 实例ID
	InstanceType string    `json:"instance_type"` // 实例类型
	UserID       string    `json:"user_id"`       // 用户ID
	AccountID    string    `json:"account_id"`    // 账户ID
	IPv4         string    `json:"ipv4"`          // 公网IPv4地址
	Region       string    `json:"region"`        // 区域
	LaunchTime   string    `json:"launch_time"`   // 启动时间
	ReportTime   string    `json:"report_time"`   // 上报时间
	LastSeen     time.Time `json:"-"`             // 最后一次上报的时间戳
}

// IPLock IP锁定信息
type IPLock struct {
	IP        string    // 锁定的IP地址
	ExpiresAt time.Time // 锁定过期时间
}

// Client 表示一个WebSocket客户端连接
type Client struct {
	Conn *websocket.Conn // WebSocket连接
	Pool *Pool           // 所属的连接池
	mu   sync.Mutex      // 互斥锁用于并发控制
}

// Pool 表示WebSocket客户端连接池
type Pool struct {
	Clients    map[*Client]bool             // 存储所有活跃的客户端连接
	Instances  map[string]*InstanceMetadata // 存储实例状态，key为实例ID
	Register   chan *Client                 // 用于注册新客户端的通道
	Unregister chan *Client                 // 用于注销客户端的通道
	mu         sync.RWMutex                 // 读写锁用于并发控制

	// 新增：IP锁定映射表
	ipLocks   map[string]*IPLock // 存储实例ID -> IP锁定信息
	ipLocksMu sync.RWMutex       // IP锁定映射表的互斥锁
}

// NewPool 创建一个新的连接池
func NewPool() *Pool {
	return &Pool{
		Clients:    make(map[*Client]bool),
		Instances:  make(map[string]*InstanceMetadata),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		ipLocks:    make(map[string]*IPLock),
	}
}

// Start 启动连接池的处理循环
func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			pool.mu.Lock()
			pool.Clients[client] = true
			pool.mu.Unlock()

		case client := <-pool.Unregister:
			pool.mu.Lock()
			delete(pool.Clients, client)
			client.Conn.Close()
			pool.mu.Unlock()
		}
	}
}

// ReadMessages 处理客户端消息
func (c *Client) ReadMessages() {
	defer func() {
		c.Pool.Unregister <- c
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("读取消息错误: %v", err)
			}
			break
		}

		// 解析实例数据
		var metadata InstanceMetadata
		if err := json.Unmarshal(message, &metadata); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 更新实例状态
		c.Pool.UpdateInstance(&metadata)
	}
}

// OnIPChangeEvent 处理IP变更事件
func (pool *Pool) OnIPChangeEvent(instanceID string, newIP string) {
	// 锁定实例的IP 30秒
	pool.LockInstanceIP(instanceID, newIP, 30*time.Second)
}

// LockInstanceIP 锁定实例IP，使用提供的IP覆盖客户机上报的IP
func (pool *Pool) LockInstanceIP(instanceID string, newIP string, duration time.Duration) {
	pool.ipLocksMu.Lock()

	// 创建或更新IP锁定记录
	pool.ipLocks[instanceID] = &IPLock{
		IP:        newIP,
		ExpiresAt: time.Now().Add(duration),
	}

	pool.ipLocksMu.Unlock()

	log.Printf("锁定实例[%s]的IP为[%s]，持续%v", instanceID, newIP, duration)

	// 立即更新实例池中的IP信息（如果实例存在）
	pool.mu.Lock()
	if instance, exists := pool.Instances[instanceID]; exists {
		oldIP := instance.IPv4
		instance.IPv4 = newIP
		log.Printf("实例[%s]IP立即更新: %s -> %s", instanceID, oldIP, newIP)
	}
	pool.mu.Unlock()
}

// UpdateInstance 更新实例状态，考虑IP锁定
func (pool *Pool) UpdateInstance(metadata *InstanceMetadata) {
	// 检查该实例是否在IP锁定状态
	var useLockedIP bool
	var lockedIP string

	pool.ipLocksMu.RLock()
	if ipLock, exists := pool.ipLocks[metadata.InstanceID]; exists {
		if time.Now().Before(ipLock.ExpiresAt) {
			// IP锁定未过期，使用锁定的IP替换上报的IP
			useLockedIP = true
			lockedIP = ipLock.IP
		} else {
			// IP锁定已过期，需要从锁定表中移除
			pool.ipLocksMu.RUnlock()
			pool.ipLocksMu.Lock()
			delete(pool.ipLocks, metadata.InstanceID)
			pool.ipLocksMu.Unlock()
			pool.ipLocksMu.RLock()
		}
	}
	pool.ipLocksMu.RUnlock()

	// 如果IP锁定生效，替换上报的IP
	if useLockedIP {
		originalIP := metadata.IPv4
		metadata.IPv4 = lockedIP
		log.Printf("实例[%s]上报的IP[%s]被锁定的IP[%s]覆盖", metadata.InstanceID, originalIP, lockedIP)
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	metadata.LastSeen = time.Now()

	if _, exists := pool.Instances[metadata.InstanceID]; !exists {
		// 新实例上线
		log.Printf("新实例上线: ID=%s, 类型=%s, 用户=%s, IP=%s, 区域=%s",
			metadata.InstanceID,
			metadata.InstanceType,
			metadata.UserID,
			metadata.IPv4,
			metadata.Region)

		// 保存实例信息
		pool.Instances[metadata.InstanceID] = metadata

		// 发送实例上线TG通知
		go func(m *InstanceMetadata) {
			db := repository.GetDB()
			err := tg.NotifyInstanceStatus(
				db,
				true, // isOnline
				m.UserID,
				m.AccountID,
				m.InstanceID,
				m.IPv4,
				m.InstanceType,
				m.Region,
			)
			if err != nil {
				log.Printf("发送实例上线TG通知失败: %v", err)
			}
		}(metadata)
	} else {
		// 更新现有实例
		pool.Instances[metadata.InstanceID] = metadata
	}
}

// CheckInstanceStatus 定期检查实例状态，同时清理过期的IP锁定
func (pool *Pool) CheckInstanceStatus() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var offlineInstances []*InstanceMetadata
		userMap := make(map[string]bool) // 用于去重的用户Map
		now := time.Now()

		// 清理过期的IP锁定
		pool.ipLocksMu.Lock()
		for instanceID, lock := range pool.ipLocks {
			if now.After(lock.ExpiresAt) {
				log.Printf("实例[%s]的IP锁定已过期，移除锁定", instanceID)
				delete(pool.ipLocks, instanceID)
			}
		}
		pool.ipLocksMu.Unlock()

		pool.mu.Lock()
		for instanceID, metadata := range pool.Instances {
			if now.Sub(metadata.LastSeen) > 60*time.Second {
				log.Printf("实例离线: 用户ID=%s, 账号ID=%s, IP=%s, 实例ID=%s",
					metadata.UserID,
					metadata.AccountID,
					metadata.IPv4,
					instanceID)
				// 保存离线实例信息用于发送通知
				offlineInstances = append(offlineInstances, metadata)
				// 将用户ID添加到Map中而不是数组，自动去重
				userMap[metadata.UserID] = true
				delete(pool.Instances, instanceID)
			}
		}
		pool.mu.Unlock()

		// 发送实例离线TG通知
		for _, metadata := range offlineInstances {
			go func(m *InstanceMetadata) {
				db := repository.GetDB()
				err := tg.NotifyInstanceStatus(
					db,
					false, // isOnline
					m.UserID,
					m.AccountID,
					m.InstanceID,
					m.IPv4,
					m.InstanceType,
					m.Region,
				)
				if err != nil {
					log.Printf("发送实例离线TG通知失败: %v", err)
				}
			}(metadata)
		}

		// 对去重后的用户列表进行检测
		for userID := range userMap {
			if result := GlobalDetector.DetectSingleUser(userID); result != nil {
				log.Printf("用户[%s]需要补机%d台", result.UserID, result.Count)
			}
		}
	}
}

// GetAllInstances 获取所有在线实例
func (pool *Pool) GetAllInstances() []*InstanceMetadata {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	instances := make([]*InstanceMetadata, 0, len(pool.Instances))
	for _, instance := range pool.Instances {
		instances = append(instances, instance)
	}
	return instances
}

// GetInstancesByUserID 获取指定用户ID的所有在线实例
func (pool *Pool) GetInstancesByUserID(userID string) []*InstanceMetadata {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	instances := make([]*InstanceMetadata, 0)
	for _, instance := range pool.Instances {
		if instance.UserID == userID {
			instances = append(instances, instance)
		}
	}

	return instances
}

// GetInstancesByUserIDAndRegion 获取指定用户ID和区域的所有在线实例
func (pool *Pool) GetInstancesByUserIDAndRegion(userID string, region string) []*InstanceMetadata {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	instances := make([]*InstanceMetadata, 0)
	for _, instance := range pool.Instances {
		if instance.UserID == userID && instance.Region == region {
			instances = append(instances, instance)
		}
	}

	return instances
}
