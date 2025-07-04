// pkg/pool/events.go
package pool

import (
	"log"
)

// AccountPoolEvent 定义账号池事件类型
type AccountPoolEvent string

const (
	// 定义事件类型常量
	AccountAdded   AccountPoolEvent = "账号添加"
	AccountReset   AccountPoolEvent = "账号重置"
	ManualReset    AccountPoolEvent = "手动重置"
	AccountDeleted AccountPoolEvent = "账号删除" // 账号删除事件
	IPChanged      AccountPoolEvent = "IP变更" // 新增IP变更事件
)

// AccountPoolListener 账号池事件监听器接口
type AccountPoolListener interface {
	OnAccountPoolEvent(event AccountPoolEvent, accountID string)
}

// IPChangeListener IP变更事件监听器接口
type IPChangeListener interface {
	OnIPChangeEvent(instanceID string, newIP string)
}

// EventManager 事件管理器，负责注册和触发事件
type EventManager struct {
	accountListeners []AccountPoolListener
	ipListeners      []IPChangeListener
}

// NewEventManager 创建新的事件管理器
func NewEventManager() *EventManager {
	return &EventManager{
		accountListeners: make([]AccountPoolListener, 0),
		ipListeners:      make([]IPChangeListener, 0),
	}
}

// RegisterAccountListener 注册账号事件监听器
func (em *EventManager) RegisterAccountListener(listener AccountPoolListener) {
	em.accountListeners = append(em.accountListeners, listener)
	log.Printf("已注册新的账号事件监听器")
}

// RegisterIPChangeListener 注册IP变更事件监听器
func (em *EventManager) RegisterIPChangeListener(listener IPChangeListener) {
	em.ipListeners = append(em.ipListeners, listener)
	log.Printf("已注册新的IP变更事件监听器")
}

// TriggerEvent 触发账号事件
func (em *EventManager) TriggerEvent(event AccountPoolEvent, accountID string) {
	log.Printf("调试: 触发事件: 类型=%s, 账号ID=%s, 监听器数量=%d",
		event, accountID, len(em.accountListeners))

	// 添加空检查
	if len(em.accountListeners) == 0 {
		log.Printf("警告: 没有注册的账号事件监听器!")
	}

	for i, listener := range em.accountListeners {
		if listener == nil {
			log.Printf("警告: 第%d个监听器为nil，跳过", i)
			continue
		}

		// 添加panic恢复
		func(l AccountPoolListener, idx int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("严重错误: 事件监听器%d处理事件时panic: %v", idx, r)
				}
			}()

			log.Printf("调试: 通知监听器%d: %T", idx, l)
			l.OnAccountPoolEvent(event, accountID)
		}(listener, i)
	}

	log.Printf("调试: 事件[%s]处理完成", event)
}

// TriggerIPChangeEvent 触发IP变更事件
func (em *EventManager) TriggerIPChangeEvent(instanceID string, newIP string) {
	log.Printf("触发IP变更事件: 实例ID=%s, 新IP=%s", instanceID, newIP)
	for _, listener := range em.ipListeners {
		listener.OnIPChangeEvent(instanceID, newIP)
	}
}

// 全局事件管理器
var (
	globalEventManager *EventManager
)

// 初始化全局事件管理器
func init() {
	globalEventManager = NewEventManager()
}

// GetEventManager 获取全局事件管理器
func GetEventManager() *EventManager {
	return globalEventManager
}
