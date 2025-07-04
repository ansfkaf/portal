// pkg/pool/detector.go
package pool

import (
	"log"
	"sync"
	"time"

	"portal/model"

	"gorm.io/gorm"
)

// DetectResult 检测结果结构体
type DetectResult struct {
	UserID string // 用户ID
	Count  int    // 需要补机的数量
	Region string // 区域代码
}

// Detector 检测器结构体
type Detector struct {
	db      *gorm.DB       // 数据库连接
	history *MakeupHistory // 补机历史记录
	userMu  sync.Map       // 用户级别的互斥锁映射
}

// NewDetector 创建新的检测器
func NewDetector(db *gorm.DB, history *MakeupHistory) *Detector {
	return &Detector{
		db:      db,
		history: history,
		userMu:  sync.Map{},
	}
}

// getUserLock 获取指定用户的互斥锁
func (d *Detector) getUserLock(userID string) *sync.Mutex {
	actual, _ := d.userMu.LoadOrStore(userID, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// DetectAllUsers 主动检测所有用户
func (d *Detector) DetectAllUsers() []DetectResult {
	// 1. 获取所有用户的监控配置
	monitors, err := model.GetAllMonitors(d.db)
	if err != nil {
		log.Printf("获取监控配置失败: %v", err)
		return nil
	}

	var results []DetectResult

	// 2. 遍历所有监控配置开启的用户
	for _, monitor := range monitors {
		// 只处理监控开启的用户
		if !monitor.IsEnabled {
			continue
		}

		// 需要检查三个区域：香港、日本、新加坡
		regions := []string{"ap-east-1", "ap-northeast-3", "ap-southeast-1"}

		for _, region := range regions {
			// 对每个用户和区域使用独立的锁
			userLock := d.getUserLock(monitor.UserID + region)
			userLock.Lock()

			// 根据区域获取对应的阈值
			var threshold int
			switch region {
			case "ap-east-1": // 香港
				threshold = monitor.Threshold
			case "ap-northeast-3": // 日本
				threshold = monitor.JpThreshold
			case "ap-southeast-1": // 新加坡
				threshold = monitor.SgThreshold
			}

			// 如果阈值为0，跳过该区域检测
			if threshold == 0 {
				userLock.Unlock()
				continue
			}

			// 3. 获取用户在指定区域的实例数
			instances := GlobalPool.GetInstancesByUserIDAndRegion(monitor.UserID, region)
			currentCount := len(instances)

			// 4. 获取用户在补机队列中的待处理任务
			makeupQueue := GetMakeupQueue()

			// 获取该用户该区域所有等待中的任务
			waitingTasks := makeupQueue.GetWaitingTasksForRegion(region)

			// 计算等待中的任务总数
			pendingMakeupCount := 0
			for _, task := range waitingTasks {
				if task.UserID == monitor.UserID {
					pendingMakeupCount += task.TotalCount - task.CompletedCount
				}
			}

			// 5. 实际需要的实例总数是阈值
			// 当前已有的实例数是：当前在线 + 待补机数量
			actualCurrentCount := currentCount + pendingMakeupCount

			// 6. 判断是否需要补机
			if actualCurrentCount < threshold {
				needCount := threshold - actualCurrentCount

				// 7. 检查是否在冷却期内（5分钟内有补机记录）
				recentCount := d.history.GetMakeupCountForRegion(monitor.UserID, region, 5*time.Minute)

				if recentCount == 0 {
					// 8. 不在冷却期内，添加补机记录
					d.history.AddMakeupRecordWithRegion(monitor.UserID, needCount, region)

					// 将任务添加到补机队列 - 每次都创建新任务
					makeupQueue.AddToQueueWithRegion(monitor.UserID, needCount, region)
					log.Printf("主动检测: 用户[%s]在区域[%s]需要补机%d台", monitor.UserID, region, needCount)

					// 将结果添加到结果列表
					results = append(results, DetectResult{
						UserID: monitor.UserID,
						Count:  needCount,
						Region: region,
					})
				} else {
					// 新增日志：记录用户在冷却期内的补机情况
					log.Printf("主动检测: 用户[%s]在区域[%s]处于冷却期内，5分钟内已补机%d台，暂不补机",
						monitor.UserID, region, recentCount)
				}
			}

			// 解锁
			userLock.Unlock()
		}
	}

	return results
}

// DetectSingleUser 被动检测单个用户
func (d *Detector) DetectSingleUser(userID string) *DetectResult {
	// 获取用户的监控配置
	monitor, err := model.GetMonitorByUserID(d.db, userID)
	if err != nil {
		log.Printf("获取用户[%s]监控配置失败: %v", userID, err)
		return nil
	}

	// 检查监控是否开启
	if !monitor.IsEnabled {
		return nil
	}

	// 增加10秒延迟 (保留原有逻辑)
	time.Sleep(10 * time.Second)

	// 需要检查三个区域：香港、日本、新加坡
	regions := []string{"ap-east-1", "ap-northeast-3", "ap-southeast-1"}

	var result *DetectResult

	// 使用一个全局锁来确保同一用户的检测不会被并发执行
	// 这可以解决同一用户在同一时间被多次检测的问题
	userLock := d.getUserLock(userID)
	userLock.Lock()
	defer userLock.Unlock()

	// 遍历所有区域进行检测
	for _, region := range regions {
		// 不再需要对每个区域单独加锁，因为已经有了用户级别的锁

		// 根据区域获取对应的阈值
		threshold := model.GetThresholdByRegion(monitor, region)

		// 如果阈值为0，跳过该区域检测
		if threshold == 0 {
			continue
		}

		// 获取用户在指定区域的实例数
		instances := GlobalPool.GetInstancesByUserIDAndRegion(userID, region)
		currentCount := len(instances)

		// 获取用户在补机队列中的待处理任务
		makeupQueue := GetMakeupQueue()

		// 获取该用户该区域所有等待中的任务
		waitingTasks := makeupQueue.GetWaitingTasksForRegion(region)

		// 计算等待中的任务总数
		pendingMakeupCount := 0
		for _, task := range waitingTasks {
			if task.UserID == userID {
				pendingMakeupCount += task.TotalCount - task.CompletedCount
			}
		}

		// 实际需要的实例总数是阈值
		// 当前已有的实例数是：当前在线 + 待补机数量
		actualCurrentCount := currentCount + pendingMakeupCount

		// 判断是否需要补机
		if actualCurrentCount < threshold {
			needCount := threshold - actualCurrentCount

			// 检查是否在冷却期内
			recentCount := d.history.GetMakeupCountForRegion(userID, region, 5*time.Minute)

			if recentCount == 0 {
				if d.history == nil {
					log.Printf("错误：用户[%s]的补机历史记录管理器为空", userID)
					continue
				}
				// 添加补机记录
				d.history.AddMakeupRecordWithRegion(userID, needCount, region)

				// 将任务添加到补机队列 - 每次都创建新任务
				makeupQueue.AddToQueueWithRegion(userID, needCount, region)
				log.Printf("被动检测: 用户[%s]在区域[%s]需要补机%d台", userID, region, needCount)

				// 仅返回第一个需要补机的区域结果
				if result == nil {
					result = &DetectResult{
						UserID: userID,
						Count:  needCount,
						Region: region,
					}
				}
			} else {
				// 新增日志：记录用户在冷却期内的补机情况
				log.Printf("被动检测: 用户[%s]在区域[%s]处于冷却期内，5分钟内已补机%d台，暂不补机",
					userID, region, recentCount)
			}
		}
	}

	return result
}
