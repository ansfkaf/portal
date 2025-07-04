// pkg/pool/accountpool.go
package pool

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"portal/model"
	"portal/repository"
)

// AccountInfo 存储在内存池中的账号信息，包含完整的账号数据
type AccountInfo struct {
	ID                   string          // 账号ID
	UserID               string          // 用户ID
	Key1                 string          // AWS Key1
	Key2                 string          // AWS Key2
	Email                *string         // 邮箱
	Password             *string         // 密码
	Quatos               *string         // 配额
	HK                   *string         // HK区状态
	VMCount              *int            // 实例数量
	Region               *string         // 区域代码
	CreateTime           *time.Time      // 创建时间
	IsSkipped            bool            // 是否需要跳过该账号（例如曾经使用失败）
	ErrorNote            string          // 错误备注，记录失败原因
	SkippedInstanceTypes map[string]bool // 标记特定实例类型是否需要跳过（例如配额用尽）
	RegionUsedCount      int             // 当前区域已使用的实例计数
}

// AccountPool 管理可用AWS账号的内存池
type AccountPool struct {
	accounts   map[string]*AccountInfo // 以ID为键的账号映射
	mutex      sync.RWMutex            // 读写锁保护并发访问
	lastUsedID string                  // 记录上次使用的账号ID，用于循环获取
}

// 全局单例实例
var (
	globalAccountPool *AccountPool
	poolOnce          sync.Once
)

// GetAccountPool 获取全局账号池实例
func GetAccountPool() *AccountPool {
	// log.Printf("调试: 请求获取全局账号池实例")
	poolOnce.Do(func() {
		log.Printf("调试: 首次初始化全局账号池")
		globalAccountPool = NewAccountPool()
		GetEventManager().RegisterAccountListener(globalAccountPool)
		err := globalAccountPool.LoadAccountsFromDB()
		if err != nil {
			log.Printf("调试: 从数据库加载账号失败: %v", err)
		} else {
			log.Printf("调试: 从数据库成功加载账号, 当前账号数: %d", globalAccountPool.Size())
		}
	})

	// 检查账号池是否为nil或大小为0
	if globalAccountPool == nil {
		log.Printf("警告: 全局账号池是nil! 尝试重新初始化")
		globalAccountPool = NewAccountPool()
		globalAccountPool.LoadAccountsFromDB()
	} else if globalAccountPool.Size() == 0 {
		log.Printf("警告: 全局账号池大小为0! 尝试重新加载")
		globalAccountPool.LoadAccountsFromDB()
	}

	// log.Printf("调试: 返回全局账号池，大小=%d，有效=%d",
	// 	globalAccountPool.Size(), globalAccountPool.AvailableSize())
	return globalAccountPool
}

// NewAccountPool 创建新的账号池
func NewAccountPool() *AccountPool {
	return &AccountPool{
		accounts:   make(map[string]*AccountInfo),
		lastUsedID: "",
	}
}

// LoadAccountsFromDB 从数据库加载所有有效账号到内存池
func (p *AccountPool) LoadAccountsFromDB() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 记录原来账号池的大小
	oldSize := len(p.accounts)

	// 清空现有账号池
	p.accounts = make(map[string]*AccountInfo)
	p.lastUsedID = ""

	// 获取数据库连接
	db := repository.GetDB()
	if db == nil {
		return nil // 数据库未初始化时不报错，返回空池
	}

	// 从数据库获取所有有效账号
	var accounts []model.Account
	err := db.Where("(quatos != '账号已失效' OR quatos IS NULL)").
		Order("id ASC"). // 按照账号ID升序排列
		Find(&accounts).Error
	if err != nil {
		return err
	}

	// 加载到内存池
	for _, account := range accounts {
		p.accounts[account.ID] = &AccountInfo{
			ID:                   account.ID,
			UserID:               account.UserID,
			Key1:                 account.Key1,
			Key2:                 account.Key2,
			Email:                account.Email,
			Password:             account.Password,
			Quatos:               account.Quatos,
			HK:                   account.HK,
			VMCount:              account.VMCount,
			Region:               account.Region,
			CreateTime:           account.CreateTime,
			IsSkipped:            false,
			ErrorNote:            "",
			SkippedInstanceTypes: make(map[string]bool), // 初始化为空映射
			RegionUsedCount:      0,                     // 初始化实例计数为0
		}
	}

	// 如果账号池大小增加了，触发账号池刷新事件
	newSize := len(p.accounts)
	if newSize > oldSize {
		log.Printf("账号池已刷新，当前账号数: %d，新增账号数: %d", newSize, newSize-oldSize)
		GetEventManager().TriggerEvent(AccountAdded, "")
	}

	return nil
}

// AddAccount 添加新账号到内存池
func (p *AccountPool) AddAccount(account model.Account) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 检查账号是否有效
	if account.Quatos != nil && *account.Quatos == "账号已失效" {
		return // 不添加无效账号
	}

	// 检查之前是否已存在该账号
	_, exists := p.accounts[account.ID]

	// 添加或更新账号
	p.accounts[account.ID] = &AccountInfo{
		ID:                   account.ID,
		UserID:               account.UserID,
		Key1:                 account.Key1,
		Key2:                 account.Key2,
		Email:                account.Email,
		Password:             account.Password,
		Quatos:               account.Quatos,
		HK:                   account.HK,
		VMCount:              account.VMCount,
		Region:               account.Region,
		CreateTime:           account.CreateTime,
		IsSkipped:            false,
		ErrorNote:            "",
		SkippedInstanceTypes: make(map[string]bool), // 初始化为空映射
		RegionUsedCount:      0,                     // 初始化实例计数为0
	}

	// 如果是新添加的账号，触发事件
	if !exists {
		GetEventManager().TriggerEvent(AccountAdded, account.ID)
		log.Printf("账号池: 添加新账号ID=%s，并触发账号添加事件", account.ID)
	}
}

// RemoveAccount 从内存池移除账号
func (p *AccountPool) RemoveAccount(accountID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.accounts, accountID)

	// 如果删除的是上次使用的账号，重置lastUsedID
	if p.lastUsedID == accountID {
		p.lastUsedID = ""
	}
}

// UpdateAccountStatus 更新账号状态
func (p *AccountPool) UpdateAccountStatus(accountID string, isValid bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 如果账号无效，从池中移除
	if !isValid {
		delete(p.accounts, accountID)
		// 如果删除的是上次使用的账号，重置lastUsedID
		if p.lastUsedID == accountID {
			p.lastUsedID = ""
		}
		return
	}

	// 如果账号有效但不在池中，暂不处理
	// 后续可以考虑从数据库加载该账号
}

// MarkAccountFailed 标记账号使用失败，记录错误信息
func (p *AccountPool) MarkAccountFailed(accountID string, errorMsg string) {
	log.Printf("调试: 准备标记账号[%s]失败，原因: %s", accountID, errorMsg)
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if account, exists := p.accounts[accountID]; exists {
		wasSkipped := account.IsSkipped
		account.IsSkipped = true
		account.ErrorNote = errorMsg
		log.Printf("调试: 账号[%s]已被标记为跳过，之前状态=%v, 当前状态=true",
			accountID, wasSkipped)
	} else {
		log.Printf("警告: 尝试标记不存在的账号[%s]失败", accountID)
	}
}

// MarkInstanceTypeFailed 标记账号对特定实例类型的使用失败
func (p *AccountPool) MarkInstanceTypeFailed(accountID string, instanceType string, errorMsg string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if account, exists := p.accounts[accountID]; exists {
		// 确保 SkippedInstanceTypes 已初始化
		if account.SkippedInstanceTypes == nil {
			account.SkippedInstanceTypes = make(map[string]bool)
		}

		// 标记此实例类型为跳过
		account.SkippedInstanceTypes[instanceType] = true

		// 更新错误信息
		account.ErrorNote = fmt.Sprintf("%s实例类型配额不足: %s", instanceType, errorMsg)

		log.Printf("账号[%s]针对实例类型[%s]被标记为跳过: %s", accountID, instanceType, errorMsg)
	}
}

// ResetAccountStatus 重置账号状态，清除跳过标记
func (p *AccountPool) ResetAccountStatus(accountID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if account, exists := p.accounts[accountID]; exists {
		// 判断是否之前是跳过状态
		wasSkipped := account.IsSkipped || len(account.SkippedInstanceTypes) > 0

		// 重置状态
		account.IsSkipped = false
		account.ErrorNote = ""
		account.SkippedInstanceTypes = make(map[string]bool) // 清空所有实例类型的跳过标记

		// 重置实例使用计数
		account.RegionUsedCount = 0

		// 如果之前为跳过状态，现在变成可用了，触发事件
		if wasSkipped {
			GetEventManager().TriggerEvent(AccountReset, accountID)
			log.Printf("账号池: 重置账号ID=%s的状态，并触发账号重置事件", accountID)
		}
	}
}

// GetNextAccountForInstanceType 获取下一个可用账号，适用于指定的实例类型和区域
func (p *AccountPool) GetNextAccountForInstanceType(instanceType string, regionCode string) *AccountInfo {
	log.Printf("调试: 开始获取实例类型[%s]区域[%s]的账号，加锁前", instanceType, regionCode)

	// 创建账号ID和错误信息的映射，用于后续锁外标记
	var needMarkAccounts = make(map[string]string)

	p.mutex.Lock()
	// log.Printf("调试: 已获取账号池互斥锁")
	defer func() {
		p.mutex.Unlock()
		// log.Printf("调试: 已释放账号池互斥锁")
	}()

	if len(p.accounts) == 0 {
		log.Printf("警告: 账号池为空！请检查数据库或加载过程")
		return nil
	}

	log.Printf("调试: 账号池当前大小=%d, 开始选择合适账号", len(p.accounts))

	// 获取所有账号ID
	ids := make([]string, 0, len(p.accounts))
	for id := range p.accounts {
		ids = append(ids, id)
	}

	// 按照ID的数值大小排序，而不是字符串字典序
	sort.Slice(ids, func(i, j int) bool {
		// 将字符串ID转换为整数进行比较
		numI, errI := strconv.Atoi(ids[i])
		numJ, errJ := strconv.Atoi(ids[j])

		// 如果转换出错（非数字ID），则退化为字符串比较
		if errI != nil || errJ != nil {
			return ids[i] < ids[j]
		}

		// 按数字大小排序
		return numI < numJ
	})

	log.Printf("调试: 已排序账号ID列表，共%d个", len(ids))

	// 计算所需实例计数
	instanceCount := getInstanceCountForType(instanceType)
	log.Printf("调试: 实例类型[%s]需要计数为%d", instanceType, instanceCount)

	// 按ID从小到大顺序，返回第一个与区域匹配且可用于指定实例类型的账号
	matchedCount := 0
	skippedCount := 0
	regionMismatchCount := 0

	for _, id := range ids {
		account := p.accounts[id]

		// 检查账号是否与请求区域匹配
		if account.Region == nil || *account.Region != regionCode {
			regionMismatchCount++
			continue
		}

		matchedCount++

		// 检查账号是否被整体跳过
		if account.IsSkipped {
			skippedCount++
			log.Printf("调试: 账号[%s]已被整体标记为跳过，原因: %s", id, account.ErrorNote)
			continue
		}

		// 检查账号是否对此实例类型被跳过
		if skipped, exists := account.SkippedInstanceTypes[instanceType]; exists && skipped {
			skippedCount++
			log.Printf("调试: 账号[%s]对实例类型[%s]被标记为跳过", id, instanceType)
			continue
		}

		// 检查区域实例使用量是否已达上限（4个实例）
		if account.RegionUsedCount+instanceCount > 4 {
			// 不直接标记账号，只记录下需要标记的账号和原因
			needMarkAccounts[account.ID] = fmt.Sprintf("%s区域配额已满（最多4个实例）", regionCode)

			log.Printf("调试: 账号[%s]在区域[%s]添加[%s]后将超过区域配额，当前使用量：%d，需要：%d，待标记",
				account.ID, regionCode, instanceType, account.RegionUsedCount, instanceCount)

			skippedCount++
			continue
		}

		// 记录日志，方便跟踪账号使用情况
		log.Printf("获取账号: ID=%s, 用户=%s, 区域=%s, 用于实例类型=%s, 当前实例使用量=%d, 将增加=%d",
			id, account.UserID, regionCode, instanceType, account.RegionUsedCount, instanceCount)
		log.Printf("调试: 成功选择账号ID=%s，符合所有条件", account.ID)

		// 在锁外标记需要标记的账号
		go func() {
			for accID, errMsg := range needMarkAccounts {
				p.MarkAccountFailed(accID, errMsg)
			}
		}()

		return account
	}

	// 如果所有账号都不匹配或被标记为跳过，返回nil
	log.Printf("没有可用的账号用于区域[%s]的实例类型[%s]，所有适用账号都被标记为跳过或区域不匹配", regionCode, instanceType)
	log.Printf("调试: 账号选择统计 - 区域匹配:%d, 被跳过:%d, 区域不匹配:%d", matchedCount, skippedCount, regionMismatchCount)

	// 在锁外标记需要标记的账号
	go func() {
		for accID, errMsg := range needMarkAccounts {
			p.MarkAccountFailed(accID, errMsg)
		}
	}()

	return nil
}

// getInstanceCountForType 根据实例类型获取实例计数（基于vCPU数量/2）
func getInstanceCountForType(instanceType string) int {
	switch instanceType {
	case "c5n.large": // 2 vCPU
		return 1
	case "c5n.xlarge": // 4 vCPU
		return 2
	case "c5n.2xlarge": // 8 vCPU
		return 4
	default:
		// 未知的实例类型，默认计为1个实例
		return 1
	}
}

// GetAllAccounts 获取所有可用账号
func (p *AccountPool) GetAllAccounts() []*AccountInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	accounts := make([]*AccountInfo, 0, len(p.accounts))
	for _, account := range p.accounts {
		accounts = append(accounts, account)
	}

	// 按照账号ID排序
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].ID < accounts[j].ID
	})

	return accounts
}

// GetAccount 根据ID获取账号信息
func (p *AccountPool) GetAccount(accountID string) *AccountInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.accounts[accountID]
}

// GetNextAccount 获取下一个可用账号，按ID数值从小到大顺序使用
func (p *AccountPool) GetNextAccount() *AccountInfo {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.accounts) == 0 {
		return nil
	}

	// 获取所有账号ID
	ids := make([]string, 0, len(p.accounts))
	for id := range p.accounts {
		ids = append(ids, id)
	}

	// 按照ID的数值大小排序，而不是字符串字典序
	sort.Slice(ids, func(i, j int) bool {
		// 将字符串ID转换为整数进行比较
		numI, errI := strconv.Atoi(ids[i])
		numJ, errJ := strconv.Atoi(ids[j])

		// 如果转换出错（非数字ID），则退化为字符串比较
		if errI != nil || errJ != nil {
			return ids[i] < ids[j]
		}

		// 按数字大小排序
		return numI < numJ
	})

	// 按ID从小到大顺序，返回第一个未被跳过的账号
	for _, id := range ids {
		account := p.accounts[id]
		if !account.IsSkipped {
			// 记录日志，方便跟踪账号使用情况
			log.Printf("获取账号: ID=%s, 用户=%s", id, account.UserID)
			return account
		}
	}

	// 如果所有账号都被标记为跳过，返回nil
	log.Printf("没有可用的账号，所有账号都被标记为跳过")
	return nil
}

// RefreshFromDB 从数据库刷新账号池
// 可以定期调用此方法保持同步
func (p *AccountPool) RefreshFromDB() error {
	return p.LoadAccountsFromDB()
}

// Size 返回账号池中可用账号的数量
func (p *AccountPool) Size() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return len(p.accounts)
}

// AvailableSize 返回可用账号的数量（未被标记为跳过的）
func (p *AccountPool) AvailableSize() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	count := 0
	for _, account := range p.accounts {
		if !account.IsSkipped {
			count++
		}
	}
	return count
}

func (p *AccountPool) GetAccountPoolInfo() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 获取所有账号信息
	accountList := make([]map[string]interface{}, 0, len(p.accounts))
	for id, account := range p.accounts {
		// 将每个账号信息转换为map
		accountInfo := map[string]interface{}{
			"id":                id,
			"user_id":           account.UserID,
			"key1":              account.Key1,
			"key2":              account.Key2,
			"is_skipped":        account.IsSkipped,
			"error_note":        account.ErrorNote,
			"region_used_count": account.RegionUsedCount,
		}

		// 处理可能为空的指针字段
		if account.Email != nil {
			accountInfo["email"] = *account.Email
		}
		if account.Password != nil {
			accountInfo["password"] = *account.Password
		}
		if account.Quatos != nil {
			accountInfo["quatos"] = *account.Quatos
		}
		if account.HK != nil {
			accountInfo["hk"] = *account.HK
		}
		if account.VMCount != nil {
			accountInfo["vm_count"] = *account.VMCount
		}
		if account.Region != nil {
			accountInfo["region"] = *account.Region
		}
		if account.CreateTime != nil {
			accountInfo["create_time"] = account.CreateTime.Format("2006-01-02 15:04:05")
		}

		accountList = append(accountList, accountInfo)
	}

	// 对账号列表按ID排序
	sort.Slice(accountList, func(i, j int) bool {
		idI, _ := accountList[i]["id"].(string)
		idJ, _ := accountList[j]["id"].(string)
		return idI < idJ
	})

	return map[string]interface{}{
		"total":     len(p.accounts),
		"available": p.AvailableSize(),
		"last_used": p.lastUsedID,
		"accounts":  accountList,
	}
}

// ResetAllAccountsStatus 重置所有账号的状态，清除所有跳过标记
func (p *AccountPool) ResetAllAccountsStatus() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 遍历所有账号，重置状态
	resetCount := 0
	resetIDs := make([]string, 0)

	for id, account := range p.accounts {
		if account.IsSkipped || account.RegionUsedCount > 0 {
			account.IsSkipped = false
			account.ErrorNote = ""
			account.SkippedInstanceTypes = make(map[string]bool)

			// 重置实例使用计数
			account.RegionUsedCount = 0

			resetCount++
			resetIDs = append(resetIDs, id)
		}
	}

	if resetCount > 0 {
		log.Printf("已重置 %d 个被标记为跳过或有实例使用的账号", resetCount)

		// 所有账号都重置完成后，触发一次手动重置事件
		GetEventManager().TriggerEvent(ManualReset, "")
		log.Printf("账号池: 批量重置账号状态，并触发手动重置事件")
	}
}

// 在accountpool.go中添加OnAccountPoolEvent方法
func (p *AccountPool) OnAccountPoolEvent(event AccountPoolEvent, accountID string) {
	switch event {
	case AccountDeleted:
		// 响应账号删除事件
		if accountID != "" {
			log.Printf("账号池: 收到账号删除事件，删除账号ID=%s", accountID)
			p.RemoveAccount(accountID)
		} else {
			// 如果accountID为空，可能是批量删除，尝试刷新整个池
			log.Printf("账号池: 收到账号删除事件，但未指定账号ID，尝试刷新账号池")
			_ = p.RefreshFromDB()
		}
	}
}

// IncrementInstanceUsage 增加账号的实例使用计数
func (p *AccountPool) IncrementInstanceUsage(accountID string, instanceType string, region string) {
	instanceCount := getInstanceCountForType(instanceType)
	log.Printf("调试: 准备增加账号[%s]的使用计数，实例类型=%s，区域=%s，增加量=%d",
		accountID, instanceType, region, instanceCount)

	// 添加账号池nil检查
	if p == nil {
		log.Printf("严重错误: AccountPool为空指针！无法增加实例使用计数")
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if account, exists := p.accounts[accountID]; exists {
		oldCount := account.RegionUsedCount
		// 验证账号区域是否与请求区域匹配
		if account.Region != nil && *account.Region == region {
			// 增加区域实例使用计数
			account.RegionUsedCount += instanceCount
			log.Printf("调试: 账号[%s]使用计数已增加: %d -> %d",
				accountID, oldCount, account.RegionUsedCount)

			// 检查是否已达到区域限制
			if account.RegionUsedCount >= 4 {
				log.Printf("调试: 账号[%s]区域[%s]使用量达到上限，准备标记为跳过", accountID, region)
				// 原有标记逻辑...
			}
		} else {
			log.Printf("警告: 账号[%s]区域不匹配, 请求区域=%s, 账号区域=%v",
				accountID, region, account.Region)
		}
	} else {
		log.Printf("警告: 尝试更新不存在的账号[%s]使用计数", accountID)
	}
}

// getCPUForInstanceType 根据实例类型获取CPU数量
func getCPUForInstanceType(instanceType string) int {
	switch instanceType {
	case "c5n.large":
		return 2
	case "c5n.xlarge":
		return 4
	default:
		// 默认情况，返回较大值以避免超额使用
		return 2
	}
}
