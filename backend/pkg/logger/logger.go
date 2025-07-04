// pkg/logger/logger.go
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 自定义日志结构
type Logger struct {
	logger *log.Logger
	writer *lumberjack.Logger
}

var (
	instance *Logger
	once     sync.Once
)

// 自动初始化
func init() {
	// 加载环境变量(如果还没加载)
	if os.Getenv("LOG_PATH") == "" && os.Getenv("MYSQL_HOST") == "" {
		// 尝试加载.env文件
		currentDir, _ := os.Getwd()
		parentDir := filepath.Dir(currentDir)
		envPath := filepath.Join(parentDir, ".env")

		if _, err := os.Stat(envPath); err == nil {
			// 读取并设置环境变量，但不覆盖已存在的变量
			if data, err := os.ReadFile(envPath); err == nil {
				lines := string(data)
				for _, line := range filepath.SplitList(lines) {
					if line == "" || line[0] == '#' {
						continue
					}
					parts := strings.SplitN(line, "=", 2)
					if len(parts) != 2 {
						continue
					}
					if os.Getenv(parts[0]) == "" {
						os.Setenv(parts[0], parts[1])
					}
				}
			}
		}
	}

	// 从环境变量获取配置
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "logs/portal.log"
	}

	maxSize, _ := strconv.Atoi(os.Getenv("LOG_MAX_SIZE"))
	if maxSize <= 0 {
		maxSize = 10 // 默认10MB
	}

	consoleOutput := true
	if os.Getenv("LOG_CONSOLE_OUTPUT") == "false" {
		consoleOutput = false
	}

	initLogger(logPath, maxSize, consoleOutput)
}

// 初始化日志器
func initLogger(logPath string, maxSize int, consoleOutput bool) {
	once.Do(func() {
		// 确保日志目录存在
		logDir := filepath.Dir(logPath)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				log.Fatalf("无法创建日志目录: %v", err)
			}
		}

		// 创建日志轮转器
		writer := &lumberjack.Logger{
			Filename: logPath,
			MaxSize:  maxSize, // 最大尺寸，单位MB
		}

		// 定义日志输出目标
		var output io.Writer = writer

		// 如果需要，同时输出到控制台
		if consoleOutput {
			output = io.MultiWriter(os.Stdout, writer)
		}

		// 创建日志器
		logger := log.New(output, "", log.LstdFlags)

		instance = &Logger{
			logger: logger,
			writer: writer,
		}

		// 替换标准日志
		log.SetOutput(output)
		log.SetFlags(log.LstdFlags)

		// 日志系统初始化完成
		Info("日志系统初始化完成，最大日志大小: %dMB", maxSize)
	})
}

// 格式化日志消息
func formatLogWithLevel(level, format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] %s", level, message)
}

// Info 记录普通日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.logger.Println(formatLogWithLevel("INFO", format, args...))
}

// Error 记录错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.logger.Println(formatLogWithLevel("ERROR", format, args...))
}

// GetLogger 获取日志实例
func GetLogger() *Logger {
	if instance == nil {
		// 这里走init时的默认配置
		initLogger("logs/portal.log", 10, true)
	}
	return instance
}

// Info 全局方法 - 记录普通日志
func Info(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

// Error 全局方法 - 记录错误日志
func Error(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

// Close 关闭日志文件
func Close() error {
	if instance != nil && instance.writer != nil {
		return instance.writer.Close()
	}
	return nil
}
