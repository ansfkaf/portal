// pkg/pool/poolint.go
package pool

import (
	"context"
	"log"
	"net/http"
	"portal/repository"
	"portal/service/instance"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// MakeupRecord 补机记录结构体
type MakeupRecord struct {
	UserID    string    // 用户ID
	Count     int       // 补机数量
	Timestamp time.Time // 补机时间
	Region    string    // 区域代码
}

// MakeupHistory 补机历史记录管理器
type MakeupHistory struct {
	records map[string][]*MakeupRecord // key为UserID，value为该用户的补机记录列表
	mu      sync.RWMutex               // 读写锁
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源的连接,生产环境需要更严格的检查
		},
	}

	// 全局变量
	GlobalPool          *Pool
	GlobalMakeupHistory *MakeupHistory
	GlobalDetector      *Detector
	GlobalIPChecker     *IPRangeChecker
	globalDB            *gorm.DB
)

// InitPool 初始化WebSocket服务
func InitPool() {
	// 从 repository 获取数据库连接
	globalDB = repository.GetDB()
	GlobalPool = NewPool()
	GlobalMakeupHistory = &MakeupHistory{
		records: make(map[string][]*MakeupRecord),
	}
	GlobalDetector = NewDetector(globalDB, GlobalMakeupHistory)

	// 向事件管理器注册IP变更事件监听器
	GetEventManager().RegisterIPChangeListener(GlobalPool)

	// 初始化账号池
	accountPool := GetAccountPool() // 确保账号池被初始化
	// 从数据库加载账号
	err := accountPool.LoadAccountsFromDB()
	if err != nil {
		log.Printf("初始化账号池失败: %v", err)
	} else {
		log.Printf("账号池初始化成功: %d个账号", accountPool.Size())
	}

	// 初始化补机队列
	GetMakeupQueue()

	// 初始化IP范围检查器 - 在这里直接初始化，不依赖外部调用
	initIPRangeChecker()

	// 启动连接池的goroutine
	go GlobalPool.Start()
	// 启动实例状态检查
	go GlobalPool.CheckInstanceStatus()
	// 启动主动检测
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			results := GlobalDetector.DetectAllUsers()
			for _, result := range results {
				log.Printf("主动检测: 用户[%s]需要补机%d台", result.UserID, result.Count)
			}
		}
	}()
}

// initIPRangeChecker 初始化IP范围检查器
func initIPRangeChecker() {
	// 创建实例服务
	instanceService := instance.NewInstanceService(globalDB)

	// 初始化IP范围检查器
	GlobalIPChecker = NewIPRangeChecker(globalDB, GlobalPool, instanceService)

	// 启动定时IP范围检查（每5分钟检查一次）
	go func() {
		// 给其他服务一些时间初始化
		time.Sleep(10 * time.Second)

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		// 减少启动日志
		log.Printf("IP范围检查服务已启动")

		for range ticker.C {
			ctx := context.Background()
			if err := GlobalIPChecker.CheckAllUsers(ctx); err != nil {
				// 只在出错时记录日志
				log.Printf("定时检查IP范围失败: %v", err)
			}
			// 移除成功完成的日志输出
		}
	}()
}

// TriggerIPRangeCheck 触发单个用户的IP范围检查
func TriggerIPRangeCheck(ctx context.Context, userID string) error {
	if GlobalIPChecker == nil {
		log.Printf("IP范围检查器未初始化，尝试重新初始化")
		initIPRangeChecker()
	}

	if GlobalIPChecker != nil {
		return GlobalIPChecker.CheckSingleUser(ctx, userID)
	}

	return nil
}

// TriggerAllIPRangeCheck 触发所有用户的IP范围检查
func TriggerAllIPRangeCheck(ctx context.Context) error {
	if GlobalIPChecker == nil {
		log.Printf("IP范围检查器未初始化，尝试重新初始化")
		initIPRangeChecker()
	}

	if GlobalIPChecker != nil {
		return GlobalIPChecker.CheckAllUsers(ctx)
	}

	return nil
}

// GetAllRecords 获取所有补机历史记录
func (mh *MakeupHistory) GetAllRecords() map[string][]*MakeupRecord {
	mh.mu.RLock()
	defer mh.mu.RUnlock()

	// 创建一个新的map来存储记录的副本
	records := make(map[string][]*MakeupRecord)
	for userID, userRecords := range mh.records {
		records[userID] = append([]*MakeupRecord{}, userRecords...)
	}

	return records
}

// HandleWebSocket 处理新的WebSocket连接
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket升级失败:", err)
		return
	}

	client := &Client{
		Conn: conn,
		Pool: GlobalPool,
	}

	GlobalPool.Register <- client

	// 启动goroutine处理客户端消息
	go client.ReadMessages()
}

// ClearAllRecords 清空所有补机历史记录
func (h *MakeupHistory) ClearAllRecords() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 清空记录，注意使用正确的类型
	h.records = make(map[string][]*MakeupRecord)
}

// GetMakeupCountForRegion 获取指定用户在指定区域和时间段内的补机总数
func (mh *MakeupHistory) GetMakeupCountForRegion(userID string, region string, duration time.Duration) int {
	mh.mu.RLock()
	defer mh.mu.RUnlock()

	key := userID + ":" + region
	records := mh.records[key]
	if len(records) == 0 {
		return 0
	}

	count := 0
	timeThreshold := time.Now().Add(-duration)

	// 从后往前遍历，因为新记录在后面
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if record.Timestamp.After(timeThreshold) {
			count += record.Count
		} else {
			// 早于阈值时间的记录可以直接跳过
			break
		}
	}

	return count
}

// AddMakeupRecordWithRegion 添加带区域的补机记录
func (mh *MakeupHistory) AddMakeupRecordWithRegion(userID string, count int, region string) {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	// 组合键
	key := userID + ":" + region

	// 检查是否在短时间内（例如1秒）有相同的记录
	records := mh.records[key]
	now := time.Now()

	// 遍历最近的记录
	for _, record := range records {
		// 如果1秒内有相同count的记录，直接返回不添加
		if now.Sub(record.Timestamp) < time.Second && record.Count == count {
			log.Printf("跳过添加重复的补机记录：用户[%s]，区域[%s]，数量[%d]", userID, region, count)
			return
		}
	}

	// 没有重复记录，正常添加
	newRecord := &MakeupRecord{
		UserID:    userID,
		Count:     count,
		Timestamp: now,
		Region:    region,
	}

	mh.records[key] = append(mh.records[key], newRecord)

	log.Printf("添加补机记录：用户[%s]，区域[%s]，数量[%d]", userID, region, count)
}
