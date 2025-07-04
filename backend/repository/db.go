// repository/db.go
package repository

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db   *gorm.DB  // 全局数据库连接实例
	once sync.Once // 确保初始化只执行一次
)

// InitDB 初始化数据库连接
func InitDB() error {
	var err error
	once.Do(func() {
		// 从环境变量中读取数据库配置
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			os.Getenv("MYSQL_USERNAME"),
			os.Getenv("MYSQL_PASSWORD"),
			os.Getenv("MYSQL_HOST"),
			os.Getenv("MYSQL_PORT"),
			os.Getenv("MYSQL_DATABASE"),
		)

		// 自定义日志配置
		newLogger := logger.New(
			log.New(os.Stdout, "", log.LstdFlags), // 使用标准日志格式
			logger.Config{
				SlowThreshold:             time.Second, // 慢 SQL 阈值
				LogLevel:                  logger.Info, // 日志级别
				IgnoreRecordNotFoundError: true,        // 忽略记录未找到错误
				Colorful:                  false,       // 禁用颜色输出
			},
		)

		// 连接数据库
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: newLogger,
		})
		if err != nil {
			err = fmt.Errorf("failed to connect database: %v", err)
			return
		}

		// 测试数据库连接
		sqlDB, err := db.DB()
		if err != nil {
			err = fmt.Errorf("failed to get database instance: %v", err)
			return
		}

		// 设置连接池配置
		sqlDB.SetMaxIdleConns(10)   // 最大空闲连接数
		sqlDB.SetMaxOpenConns(500)  // 最大打开连接数
		sqlDB.SetConnMaxLifetime(0) // 连接的最大存活时间（0 表示不限制）
	})

	return err
}

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	return db
}
