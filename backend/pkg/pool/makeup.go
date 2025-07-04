// pkg/pool/makeup.go
package pool

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// MakeupQueueItem 补机队列项
type MakeupQueueItem struct {
	UserID         string    // 用户ID
	Region         string    // 区域代码
	TotalCount     int       // 需要补机总数
	CompletedCount int       // 已完成数量
	AddTime        time.Time // 添加到队列的时间
	Status         string    // 状态：等待中、进行中、已完成
	QueueID        string    // 队列项唯一ID，格式为：userID:region:timestamp
}

// MakeupQueue 补机队列管理器
type MakeupQueue struct {
	queue       map[string]*MakeupQueueItem // 任务队列键 -> 补机任务
	mu          sync.RWMutex                // 读写锁
	taskChannel chan string                 // 任务通知通道
	isRunning   bool                        // 是否已启动处理循环
	processing  bool                        // 是否正在处理任务的标志
}

// 全局补机队列
var (
	globalMakeupQueue *MakeupQueue
	makeupQueueOnce   sync.Once
)

// GetMakeupQueue 获取全局补机队列实例
func GetMakeupQueue() *MakeupQueue {
	makeupQueueOnce.Do(func() {
		globalMakeupQueue = &MakeupQueue{
			queue:       make(map[string]*MakeupQueueItem),
			taskChannel: make(chan string, 100), // 缓冲区大小设为100，避免阻塞
			isRunning:   false,
			processing:  false,
		}
		// 启动补机处理协程
		go globalMakeupQueue.StartProcessing()

		// 启动账号重置协程
		go globalMakeupQueue.ResetHKRequestingAccounts()

		// 注册为账号池事件的监听器
		GetEventManager().RegisterAccountListener(globalMakeupQueue)
		log.Printf("补机队列已注册为账号池事件监听器")
	})
	return globalMakeupQueue
}

// 在 OnAccountPoolEvent 方法中增加以下内容
func (mq *MakeupQueue) OnAccountPoolEvent(event AccountPoolEvent, accountID string) {
	log.Printf("补机队列收到事件: %s, 账号ID: %s", event, accountID)

	// 根据不同事件类型处理
	switch event {
	case AccountAdded, AccountReset, ManualReset:
		// 这些事件都应该触发重置卡住的任务
		mq.ResetStuckTasks()

		// 主动处理等待中的任务
		mq.processExistingTasks()
	}
}

// ResetStuckTasks 将所有"进行中"但未完成的任务重置为"等待中"
func (mq *MakeupQueue) ResetStuckTasks() {
	mq.mu.Lock()

	// 寻找所有状态为"进行中"但未完成的任务
	stuckTasks := make([]string, 0)
	for key, task := range mq.queue {
		if task.Status == "进行中" && task.CompletedCount < task.TotalCount {
			// 将任务状态重置为"等待中"
			task.Status = "等待中"
			stuckTasks = append(stuckTasks, key)

			log.Printf("重置任务: 用户[%s]，区域[%s]，总数=%d, 已完成=%d, 队列ID=%s",
				task.UserID, task.Region, task.TotalCount, task.CompletedCount, task.QueueID)
		}
	}

	// 重置处理标志，确保可以处理新任务
	mq.processing = false

	mq.mu.Unlock()

	// 将重置的任务重新放入处理队列
	if len(stuckTasks) > 0 {
		log.Printf("开始重新处理%d个任务", len(stuckTasks))

		// 清空任务通道，确保没有旧任务卡在那里
		for len(mq.taskChannel) > 0 {
			<-mq.taskChannel
		}

		// 将任务重新加入队列并立即开始处理
		for _, taskKey := range stuckTasks {
			select {
			case mq.taskChannel <- taskKey:
				log.Printf("已将任务[%s]重新加入处理队列", taskKey)
			default:
				log.Printf("任务通知通道已满，任务[%s]未能重新加入队列", taskKey)
			}
		}
	}
}

// StartProcessing 启动处理循环
func (mq *MakeupQueue) StartProcessing() {
	mq.isRunning = true
	log.Printf("补机队列处理器启动")

	// 处理所有已有任务
	mq.processExistingTasks()

	// 启动定期检查任务状态的协程
	go mq.periodicTaskCheck()

	// 等待新任务通知
	for queueKey := range mq.taskChannel {
		// 如果已经在处理任务，记录并跳过，但不要丢弃任务
		if mq.processing {
			log.Printf("正在处理其他任务，任务[%s]将稍后处理", queueKey)
			// 将任务重新加入队列而不是直接忽略
			go func(key string) {
				time.Sleep(10 * time.Second) // 等待一段时间后重试
				mq.taskChannel <- key
				log.Printf("已将延迟的任务[%s]重新加入处理队列", key)
			}(queueKey)
			continue
		}

		// 记录日志，方便跟踪任务处理流程
		log.Printf("收到任务处理通知: %s", queueKey)

		// 通过任务键获取任务
		task := mq.GetQueueItemByKey(queueKey)
		if task == nil {
			log.Printf("任务[%s]不存在", queueKey)
			continue
		}

		// 检查任务是否需要处理
		if task.Status != "等待中" {
			log.Printf("任务[%s]状态为[%s]，不需要处理", queueKey, task.Status)
			continue
		}

		if task.CompletedCount >= task.TotalCount {
			log.Printf("任务[%s]已完成（已完成=%d, 总数=%d）",
				queueKey, task.CompletedCount, task.TotalCount)

			// 更新状态为已完成
			mq.updateTaskStatusByKey(queueKey, "已完成")
			continue
		}

		// 添加安全检查：确保任务确实有需要处理的内容
		needToProcess := task.TotalCount - task.CompletedCount
		if needToProcess <= 0 {
			log.Printf("任务[%s]没有需要处理的内容：总量=%d，已完成=%d",
				queueKey, task.TotalCount, task.CompletedCount)

			// 更新状态为已完成
			mq.updateTaskStatusByKey(queueKey, "已完成")
			continue
		}

		// 更新任务状态为进行中
		mq.updateTaskStatusByKey(queueKey, "进行中")
		log.Printf("开始处理任务[%s]，需处理%d台", queueKey, needToProcess)

		// 设置处理标志
		mq.processing = true

		// 添加处理超时保护
		processingTimer := time.AfterFunc(30*time.Minute, func() {
			if mq.processing {
				log.Printf("警告：任务[%s]处理超时(30分钟)，强制重置处理状态", queueKey)
				mq.processing = false
				// 将任务状态重置为等待中
				mq.updateTaskStatusByKey(queueKey, "等待中")
				// 重新加入队列
				mq.taskChannel <- queueKey
			}
		})

		// 使用匿名函数和defer确保无论如何processing标志都会被重置
		func() {
			defer func() {
				// 取消处理超时计时器
				processingTimer.Stop()

				// 无论任务处理是否成功，都重置处理标志
				mq.processing = false

				// 恢复可能的panic
				if r := recover(); r != nil {
					log.Printf("补机任务处理过程中发生panic: %v", r)
					// 将任务状态重置为等待中
					mq.updateTaskStatusByKey(queueKey, "等待中")
				}

				log.Printf("任务[%s]处理完成或中断", queueKey)
			}()

			// 处理任务
			remainingCount := task.TotalCount - task.CompletedCount
			err := mq.processMakeup(queueKey, remainingCount)
			if err != nil {
				log.Printf("任务[%s]处理出错: %v", queueKey, err)

				// 如果不是因为账号不足，则将状态设回等待中，以便下次处理
				if err.Error() != "没有可用的账号" {
					mq.updateTaskStatusByKey(queueKey, "等待中")
					// 将任务重新加入队列，延迟5秒再处理
					go func(key string) {
						time.Sleep(5 * time.Second)
						mq.taskChannel <- key
					}(queueKey)
				} else {
					log.Printf("由于没有可用账号，任务[%s]将保持等待状态，15分钟后重试", queueKey)
					// 即使没有可用账号，也设置一个较长的延迟后重试，避免任务永久卡住
					go func(key string) {
						time.Sleep(15 * time.Minute)
						task := mq.GetQueueItemByKey(key)
						if task != nil && task.Status == "等待中" && task.CompletedCount < task.TotalCount {
							log.Printf("定时重试缺少账号的任务[%s]", key)
							mq.taskChannel <- key
						}
					}(queueKey)
				}
			}
		}()
	}
}

// 处理已有的待处理任务
func (mq *MakeupQueue) processExistingTasks() {
	tasks := mq.GetWaitingTasks()
	if len(tasks) > 0 {
		for _, task := range tasks {
			mq.taskChannel <- task.QueueID
			log.Printf("添加任务到队列: 用户[%s]，区域[%s]，任务键[%s]",
				task.UserID, task.Region, task.QueueID)
		}
	}
}

// 生成队列ID
func generateQueueID(userID string, region string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s:%s:%d", userID, region, timestamp)
}

// AddToQueueWithRegion 添加带区域的补机任务到队列
// 强制新建任务而不是更新现有任务
func (mq *MakeupQueue) AddToQueueWithRegion(userID string, count int, region string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	now := time.Now()

	// 生成新的队列ID，确保每次都是新建任务
	queueID := generateQueueID(userID, region)

	// 创建新任务
	mq.queue[queueID] = &MakeupQueueItem{
		UserID:         userID,
		Region:         region,
		TotalCount:     count,
		CompletedCount: 0,
		AddTime:        now,
		Status:         "等待中",
		QueueID:        queueID,
	}

	log.Printf("为用户[%s]在区域[%s]创建新补机任务：数量[%d]，队列ID[%s]",
		userID, region, count, queueID)

	// 发送通知到处理通道
	go func() {
		select {
		case mq.taskChannel <- queueID:
			log.Printf("已向处理通道发送任务通知：[%s]", queueID)
		default:
			log.Printf("任务通知通道已满，任务[%s]无法通知，将启动恢复程序", queueID)
			// 如果通道已满，尝试在短暂延迟后重发
			go func() {
				time.Sleep(3 * time.Second) // 等待片刻
				// 重新尝试发送此任务
				select {
				case mq.taskChannel <- queueID:
					log.Printf("恢复程序已将任务[%s]重新发送到处理通道", queueID)
				default:
					log.Printf("处理通道仍然满，任务[%s]无法处理，稍后将由定期检查机制处理", queueID)
				}
			}()
		}
	}()
}

// 移除不再使用的原有方法
// AddToQueue 原始方法不再使用，保留为空函数以保持兼容性
func (mq *MakeupQueue) AddToQueue(userID string, count int) {
	// 转发到新方法，使用香港区域作为默认
	mq.AddToQueueWithRegion(userID, count, "ap-east-1")
}

// GetQueueItemByKey 通过队列键获取任务
func (mq *MakeupQueue) GetQueueItemByKey(queueKey string) *MakeupQueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	return mq.queue[queueKey]
}

// updateTaskStatusByKey 通过队列键更新任务状态
func (mq *MakeupQueue) updateTaskStatusByKey(queueKey string, status string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if item, exists := mq.queue[queueKey]; exists {
		item.Status = status
		log.Printf("更新任务[%s]状态为[%s]", queueKey, status)
	}
}

// IncrementCompletedCount 根据queueKey增加已完成数量
func (mq *MakeupQueue) IncrementCompletedCount(queueKey string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if item, exists := mq.queue[queueKey]; exists {
		item.CompletedCount++
		log.Printf("任务[%s]完成计数增加: %d/%d", queueKey, item.CompletedCount, item.TotalCount)

		// 检查是否已完成所有补机
		if item.CompletedCount >= item.TotalCount {
			item.Status = "已完成"
			log.Printf("任务[%s]已全部完成，共完成[%d]台", queueKey, item.CompletedCount)
		}
	}
}

// GetWaitingTasks 获取所有等待中的任务（按添加时间排序）
func (mq *MakeupQueue) GetWaitingTasks() []*MakeupQueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	var waitingTasks []*MakeupQueueItem
	for _, task := range mq.queue {
		if task.Status == "等待中" && task.CompletedCount < task.TotalCount {
			waitingTasks = append(waitingTasks, task)
		}
	}

	// 按添加时间排序
	sort.Slice(waitingTasks, func(i, j int) bool {
		return waitingTasks[i].AddTime.Before(waitingTasks[j].AddTime)
	})

	return waitingTasks
}

// GetQueue 获取整个补机队列（按添加时间排序）
func (mq *MakeupQueue) GetQueue() []*MakeupQueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	// 将map转为切片
	items := make([]*MakeupQueueItem, 0, len(mq.queue))
	for _, item := range mq.queue {
		items = append(items, item)
	}

	// 按添加时间正序排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].AddTime.Before(items[j].AddTime)
	})

	return items
}

// processMakeup 处理用户的补机任务
func (mq *MakeupQueue) processMakeup(queueKey string, count int) error {
	task := mq.GetQueueItemByKey(queueKey)
	if task == nil {
		return fmt.Errorf("任务不存在")
	}

	userID := task.UserID
	region := task.Region

	log.Printf("调试: 开始处理补机任务[%s]，用户[%s]，区域[%s]，计划补机数量=%d",
		queueKey, userID, region, count)

	// 获取账号池状态
	accountPool := GetAccountPool()
	if accountPool == nil {
		log.Printf("严重错误: 获取账号池返回nil，补机无法继续")
		return fmt.Errorf("账号池无效")
	}

	log.Printf("调试: 补机前账号池状态: 总数=%d, 可用=%d",
		accountPool.Size(), accountPool.AvailableSize())

	// 处理计数
	processedCount := 0

	// 最大重试次数（等于账号池中账号数量或设定上限）
	maxRetries := accountPool.Size()
	if maxRetries > 10 {
		maxRetries = 10 // 限制最大重试次数为10
	}
	log.Printf("调试: 设置最大重试次数=%d", maxRetries)

	// 重试计数
	retryCount := 0

	// 循环处理每台需要补的机器
	for processedCount < count {
		// 检查是否达到最大重试次数
		if retryCount >= maxRetries {
			log.Printf("用户[%s]在区域[%s]补机失败，已达到最大重试次数(%d)，暂停任务",
				userID, region, maxRetries)
			// 将任务重置为等待状态，以便稍后重试
			mq.updateTaskStatusByKey(queueKey, "等待中")
			log.Printf("调试: 达到最大重试次数，任务状态已重置为等待中")
			return fmt.Errorf("已达到最大重试次数，暂停任务")
		}

		// 再次检查账号池状态
		log.Printf("调试: 准备为用户[%s]在区域[%s]创建实例，当前已处理=%d/%d, 重试次数=%d",
			userID, region, processedCount, count, retryCount)

		// 使用 makeupvm.go 中的函数创建实例，传递区域参数
		result, err := CreateInstanceForUser(userID, region)

		if err != nil {
			log.Printf("用户[%s]在区域[%s]补机尝试失败：%v", userID, region, err)
			retryCount++

			log.Printf("调试: 创建实例失败，当前重试次数=%d/%d, 错误=%v",
				retryCount, maxRetries, err)

			// 如果是因为没有可用账号，中断处理
			if err.Error() == "没有可用的账号" {
				// 将任务重置为等待状态，以便稍后重试
				mq.updateTaskStatusByKey(queueKey, "等待中")
				log.Printf("调试: 由于没有可用账号，任务已重置为等待中")

				// 检查账号池状态
				log.Printf("调试: 当前账号池状态: 总数=%d, 可用=%d",
					accountPool.Size(), accountPool.AvailableSize())

				if accountPool.Size() == 0 {
					log.Printf("严重错误: 账号池为空！尝试重新加载")
					accountPool.LoadAccountsFromDB()
				}

				return fmt.Errorf("没有可用的账号")
			}

			// 小延迟，避免快速重试
			log.Printf("调试: 等待2秒后重试")
			time.Sleep(2 * time.Second)
			continue
		}

		// 创建成功，增加已完成计数
		processedCount++
		mq.IncrementCompletedCount(queueKey)

		// 记录开机成功信息
		log.Printf("用户[%s]在区域[%s]补机成功，实例ID[%s]", userID, region, result.InstanceID)
		log.Printf("调试: 实例创建成功，已处理数量=%d/%d", processedCount, count)

		// 重置重试计数
		retryCount = 0

		// 小延迟，避免请求过快
		log.Printf("调试: 等待2秒后处理下一台")
		time.Sleep(2 * time.Second)
	}

	log.Printf("用户[%s]在区域[%s]本轮补机完成，共处理[%d]台", userID, region, processedCount)
	log.Printf("调试: 补机任务[%s]已全部完成，处理=%d/%d", queueKey, processedCount, count)

	// 更新任务状态为已完成
	mq.updateTaskStatusByKey(queueKey, "已完成")

	return nil
}

// GetQueueStatus 获取队列状态摘要
func (mq *MakeupQueue) GetQueueStatus() map[string]interface{} {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	activeCount := 0
	waitingCount := 0
	completedCount := 0
	totalMachines := 0
	completedMachines := 0

	for _, item := range mq.queue {
		totalMachines += item.TotalCount
		completedMachines += item.CompletedCount

		if item.Status == "进行中" {
			activeCount++
		} else if item.Status == "等待中" {
			waitingCount++
		} else if item.Status == "已完成" {
			completedCount++
		}
	}

	return map[string]interface{}{
		"queue_size":         len(mq.queue),
		"active_tasks":       activeCount,
		"waiting_tasks":      waitingCount,
		"completed_tasks":    completedCount,
		"total_machines":     totalMachines,
		"completed_machines": completedMachines,
	}
}

// ResetHKRequestingAccounts 定期重置申请HK区中的账号
func (mq *MakeupQueue) ResetHKRequestingAccounts() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		accountPool := GetAccountPool()
		accounts := accountPool.GetAllAccounts()
		resetCount := 0

		for _, account := range accounts {
			// 只重置被标记为跳过且错误原因是申请HK区相关的账号
			if account.IsSkipped && (strings.Contains(account.ErrorNote, "HK区域未开通") ||
				strings.Contains(account.ErrorNote, "HK区资源验证中")) {
				accountPool.ResetAccountStatus(account.ID)
				resetCount++
			}
		}

		if resetCount > 0 {
			log.Printf("已重置%d个申请HK区中的账号", resetCount)

			// 如果有等待中的任务，尝试重新处理
			waitingTasks := mq.GetWaitingTasks()
			if len(waitingTasks) > 0 && resetCount > 0 {
				for _, task := range waitingTasks {
					select {
					case mq.taskChannel <- task.QueueID:
						// 已通知重新处理任务
					default:
						// 通道已满，忽略
					}
				}
			}
		}
	}
}

// ClearAllQueue 清空所有补机队列
func (mq *MakeupQueue) ClearAllQueue() {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	// 清空队列
	mq.queue = make(map[string]*MakeupQueueItem)

	// 清空任务通道
	for len(mq.taskChannel) > 0 {
		<-mq.taskChannel
	}

	// 重置处理标志
	mq.processing = false

	log.Printf("已清空所有补机队列")
}

// GetWaitingTasksForRegion 获取指定区域所有等待中的任务（按添加时间排序）
func (mq *MakeupQueue) GetWaitingTasksForRegion(region string) []*MakeupQueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	var waitingTasks []*MakeupQueueItem
	for _, task := range mq.queue {
		// 检查任务是否属于指定区域
		if task.Region == region && task.Status == "等待中" && task.CompletedCount < task.TotalCount {
			waitingTasks = append(waitingTasks, task)
		}
	}

	// 按添加时间排序
	sort.Slice(waitingTasks, func(i, j int) bool {
		return waitingTasks[i].AddTime.Before(waitingTasks[j].AddTime)
	})

	return waitingTasks
}

// periodicTaskCheck 定期检查处于等待状态的任务并尝试推动处理
func (mq *MakeupQueue) periodicTaskCheck() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// log.Printf("开始执行定期任务检查...")

		waitingTasks := mq.GetWaitingTasks()
		if len(waitingTasks) == 0 {
			// log.Printf("当前没有等待中的任务")
			continue
		}

		log.Printf("发现%d个等待中的任务", len(waitingTasks))

		// 检查是否有长时间等待的任务
		now := time.Now()
		for _, task := range waitingTasks {
			waitTime := now.Sub(task.AddTime)

			// 如果任务等待超过15分钟，主动推送到处理通道
			if waitTime > 15*time.Minute {
				log.Printf("任务[%s:%s]已等待%.1f分钟，主动推送处理",
					task.UserID, task.Region, waitTime.Minutes())

				// 使用非阻塞方式尝试推送
				select {
				case mq.taskChannel <- task.QueueID:
					log.Printf("已将长时间等待的任务[%s]推送至处理通道", task.QueueID)
				default:
					log.Printf("处理通道已满，无法推送任务[%s]", task.QueueID)
				}
			}
		}

		// 检查处理标志是否异常地长时间为true
		if mq.processing {
			log.Printf("发现处理标志仍为true，这可能阻止其他任务处理，强制重置处理标志")
			mq.processing = false
		}

		log.Printf("定期任务检查完成")
	}
}
