// pkg/tg/tg.go
package tg

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"portal/model" // 使用项目的正确导入路径
	"portal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// MessageType 消息类型枚举
type MessageType string

const (
	InstanceOffline MessageType = "INSTANCE_OFFLINE" // 实例离线通知
	InstanceOnline  MessageType = "INSTANCE_ONLINE"  // 实例上线通知
)

// TgClient Telegram客户端结构体
type TgClient struct {
	bot             *tgbotapi.BotAPI
	token           string
	bindingCodes    map[string]*BindingCode
	bindingCodeLock sync.Mutex
	db              *gorm.DB
}

// BindingCode 表示一个绑定码及其关联信息
type BindingCode struct {
	Code      string    // 绑定码
	UserID    string    // 用户ID
	CreatedAt time.Time // 创建时间
	ExpiresAt time.Time // 过期时间
}

var (
	client  *TgClient
	once    sync.Once
	initErr error
)

// InitTgClient 初始化TG客户端单例
func InitTgClient() error {
	once.Do(func() {
		token := os.Getenv("TG_BOT_TOKEN")
		if token == "" {
			initErr = fmt.Errorf("TG_BOT_TOKEN 环境变量未设置")
			return
		}

		bot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			initErr = fmt.Errorf("初始化Telegram Bot失败: %v", err)
			return
		}

		client = &TgClient{
			bot:          bot,
			token:        token,
			bindingCodes: make(map[string]*BindingCode),
			db:           repository.GetDB(),
		}

		// 启动消息监听
		go client.startMessageListening()
	})
	return initErr
}

// GetClient 获取TG客户端实例
func GetClient() (*TgClient, error) {
	if client == nil {
		return nil, fmt.Errorf("Telegram客户端尚未初始化，请先调用InitTgClient()")
	}
	return client, nil
}

// getMessageTemplate 根据消息类型生成消息模板
func getMessageTemplate(msgType MessageType, userID string, accountID string, instanceID string, ipv4 string, instanceType string, region string) string {
	templates := map[MessageType]string{
		InstanceOffline: fmt.Sprintf("⚠️ *实例离线通知*\n"+
			"*账号ID*: `%s`\n"+
			"*IP地址*: %s",
			accountID, ipv4),

		InstanceOnline: fmt.Sprintf("✅ *实例上线通知*\n"+
			"*账号ID*: `%s`\n"+
			"*IP地址*: %s",
			accountID, ipv4),
	}

	if template, exists := templates[msgType]; exists {
		return template
	}

	// 默认模板
	return fmt.Sprintf("📢 *通知*\n*账号ID*: %s\n*IP地址*: %s",
		accountID, ipv4)
}

// SendInstanceStatusNotification 发送实例状态通知
func (c *TgClient) SendInstanceStatusNotification(tgUserID string, msgType MessageType, userID string, accountID string, instanceID string, ipv4 string, instanceType string, region string) error {
	if tgUserID == "" {
		return fmt.Errorf("TG用户ID为空，无法发送通知")
	}

	// 将tgUserID转换为int64
	chatID, err := strconv.ParseInt(tgUserID, 10, 64)
	if err != nil {
		return fmt.Errorf("TG用户ID格式不正确: %v", err)
	}

	// 获取消息模板
	messageText := getMessageTemplate(msgType, userID, accountID, instanceID, ipv4, instanceType, region)

	// 创建并发送消息
	telegramMsg := tgbotapi.NewMessage(chatID, messageText)
	telegramMsg.ParseMode = tgbotapi.ModeMarkdown

	// 发送消息
	_, err = c.bot.Send(telegramMsg)

	if err != nil {
		return fmt.Errorf("发送Telegram消息失败: %v", err)
	}

	return nil
}

// SendInstanceOfflineNotification 直接发送实例离线通知
func (c *TgClient) SendInstanceOfflineNotification(tgUserID string, userID string, accountID string, instanceID string, ipv4 string, instanceType string, region string) error {
	return c.SendInstanceStatusNotification(tgUserID, InstanceOffline, userID, accountID, instanceID, ipv4, instanceType, region)
}

// SendInstanceOnlineNotification 直接发送实例上线通知
func (c *TgClient) SendInstanceOnlineNotification(tgUserID string, userID string, accountID string, instanceID string, ipv4 string, instanceType string, region string) error {
	return c.SendInstanceStatusNotification(tgUserID, InstanceOnline, userID, accountID, instanceID, ipv4, instanceType, region)
}

// NotifyInstanceStatus 通知实例状态变化（检查用户设置）
func NotifyInstanceStatus(db *gorm.DB, isOnline bool, userID string, accountID string, instanceID string, ipv4 string, instanceType string, region string) error {
	// 如果客户端未初始化，则尝试初始化
	if client == nil {
		if err := InitTgClient(); err != nil {
			return fmt.Errorf("TG客户端初始化失败: %v", err)
		}
	}

	// 获取用户的TG通知设置
	isTgEnabled, tgUserID, err := model.GetTgNotificationSettings(db, userID)
	if err != nil {
		return fmt.Errorf("获取用户TG通知设置失败: %v", err)
	}

	// 检查是否启用TG通知和TG用户ID是否为空
	if !isTgEnabled || tgUserID == "" {
		// 不需要发送通知
		return nil
	}

	// 根据实例状态发送不同的通知
	if isOnline {
		err = client.SendInstanceOnlineNotification(tgUserID, userID, accountID, instanceID, ipv4, instanceType, region)
	} else {
		err = client.SendInstanceOfflineNotification(tgUserID, userID, accountID, instanceID, ipv4, instanceType, region)
	}

	if err != nil {
		return fmt.Errorf("发送TG通知失败: %v", err)
	}

	return nil
}

// GetBotInfo 获取Bot的基本信息
func (c *TgClient) GetBotInfo() tgbotapi.User {
	return c.bot.Self
}

// generateBindingCode 生成8位随机数字码
func generateBindingCode() (string, error) {
	const digits = "0123456789"
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = digits[b%byte(len(digits))]
	}
	return string(bytes), nil
}

// CreateBindingCode 为用户创建绑定码
func (c *TgClient) CreateBindingCode(userID string) (string, error) {
	c.bindingCodeLock.Lock()
	defer c.bindingCodeLock.Unlock()

	// 清理过期的绑定码
	c.cleanExpiredBindingCodes()

	// 检查用户是否已经有未过期的绑定码
	for _, code := range c.bindingCodes {
		if code.UserID == userID && time.Now().Before(code.ExpiresAt) {
			return code.Code, nil
		}
	}

	// 生成新的绑定码
	bindingCode, err := generateBindingCode()
	if err != nil {
		return "", fmt.Errorf("生成绑定码失败: %v", err)
	}

	// 存储绑定码
	now := time.Now()
	c.bindingCodes[bindingCode] = &BindingCode{
		Code:      bindingCode,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(1 * time.Hour), // 1小时有效期
	}

	return bindingCode, nil
}

// cleanExpiredBindingCodes 清理过期的绑定码
func (c *TgClient) cleanExpiredBindingCodes() {
	now := time.Now()
	for code, info := range c.bindingCodes {
		if now.After(info.ExpiresAt) {
			delete(c.bindingCodes, code)
		}
	}
}

// startMessageListening 启动消息监听
func (c *TgClient) startMessageListening() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		msg := update.Message
		if !msg.IsCommand() && strings.HasPrefix(msg.Text, "/bind/") {
			go c.handleBindCommand(msg)
		}
	}
}

// handleBindCommand 处理绑定命令
func (c *TgClient) handleBindCommand(message *tgbotapi.Message) {
	// 提取绑定码
	parts := strings.Split(message.Text, "/")
	if len(parts) != 3 {
		c.sendReply(message.Chat.ID, "无效的绑定命令格式。请使用 /bind/您的绑定码")
		return
	}

	bindingCode := parts[2]
	if len(bindingCode) != 8 {
		c.sendReply(message.Chat.ID, "无效的绑定码长度。绑定码应为8位数字。")
		return
	}

	// 验证绑定码
	c.bindingCodeLock.Lock()
	codeInfo, exists := c.bindingCodes[bindingCode]
	c.bindingCodeLock.Unlock()

	if !exists {
		c.sendReply(message.Chat.ID, "无效的绑定码。请确认您输入的绑定码正确或重新生成绑定码。")
		return
	}

	if time.Now().After(codeInfo.ExpiresAt) {
		c.sendReply(message.Chat.ID, "绑定码已过期。请重新生成绑定码。")
		return
	}

	// 将TG用户ID绑定到用户
	tgUserID := strconv.FormatInt(message.From.ID, 10)
	err := model.UpdateUserTgID(c.db, codeInfo.UserID, tgUserID)
	if err != nil {
		c.sendReply(message.Chat.ID, "绑定失败: "+err.Error())
		return
	}

	// 绑定成功，从映射中删除绑定码
	c.bindingCodeLock.Lock()
	delete(c.bindingCodes, bindingCode)
	c.bindingCodeLock.Unlock()

	c.sendReply(message.Chat.ID, "绑定成功！您将开始接收实例状态通知。")
}

// sendReply 发送回复消息
func (c *TgClient) sendReply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := c.bot.Send(msg)
	if err != nil {
		// 静默处理错误
	}
}

// SendSimpleMessage 发送简单消息
func (c *TgClient) SendSimpleMessage(tgUserID string, message string) error {
	if tgUserID == "" {
		return fmt.Errorf("TG用户ID为空，无法发送消息")
	}

	// 将tgUserID转换为int64
	chatID, err := strconv.ParseInt(tgUserID, 10, 64)
	if err != nil {
		return fmt.Errorf("TG用户ID格式不正确: %v", err)
	}

	// 创建并发送消息
	telegramMsg := tgbotapi.NewMessage(chatID, message)
	_, err = c.bot.Send(telegramMsg)
	if err != nil {
		return fmt.Errorf("发送Telegram消息失败: %v", err)
	}

	return nil
}
