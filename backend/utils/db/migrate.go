// utils/db/migrate.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"portal/model"
)

// 数据库配置
type DBConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
}

// 列信息
type Column struct {
	Name     string `gorm:"column:COLUMN_NAME"`
	DataType string `gorm:"column:COLUMN_TYPE"`
	IsNull   string `gorm:"column:IS_NULLABLE"`
}

// 管理员用户配置
type AdminConfig struct {
	Email    string
	Password string
	IsAdmin  uint8
}

// 加载环境变量
func loadConfig() (*DBConfig, *AdminConfig, error) {
	currentDir, _ := os.Getwd()
	envPath := filepath.Join(filepath.Dir(currentDir), ".env")

	// 尝试加载.env文件，如果不存在则使用环境变量
	_ = godotenv.Load(envPath)

	dbConfig := &DBConfig{
		Host:     getEnvWithDefault("MYSQL_HOST", "localhost"),
		Port:     getEnvWithDefault("MYSQL_PORT", "3306"),
		Database: getEnvWithDefault("MYSQL_DATABASE", "portal"),
		Username: getEnvWithDefault("MYSQL_USERNAME", "root"),
		Password: getEnvWithDefault("MYSQL_PASSWORD", ""),
	}

	adminConfig := &AdminConfig{
		Email:    getEnvWithDefault("ADMIN_EMAIL", "admin@qq.com"),
		Password: getEnvWithDefault("ADMIN_PASSWORD", "Aa112233"),
		IsAdmin:  1,
	}

	return dbConfig, adminConfig, nil
}

// 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// 连接数据库
func connectDB(config *DBConfig, dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Username, config.Password, config.Host, config.Port, dbName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// 创建管理员用户
func createAdminUser(db *gorm.DB, adminConfig *AdminConfig) error {
	// 检查是否已存在管理员用户
	var count int64
	db.Model(&model.User{}).Where("email = ? OR is_admin = 1", adminConfig.Email).Count(&count)

	if count > 0 {
		fmt.Println("管理员用户已存在，跳过创建")
		return nil
	}

	// 创建管理员用户
	admin := model.User{
		Email:    adminConfig.Email,
		Password: adminConfig.Password, // 模型的BeforeCreate钩子会自动处理密码加密
		IsAdmin:  adminConfig.IsAdmin,
	}

	// 创建用户 - model.User的AfterCreate钩子会自动创建关联表的默认记录
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("创建管理员用户失败: %v", err)
	}

	fmt.Printf("管理员用户 '%s' 创建成功\n", adminConfig.Email)
	return nil
}

func main() {
	// 解析命令行参数
	createAdmin := flag.Bool("create-admin", true, "创建管理员用户")
	flag.Parse()

	// 1. 加载配置
	dbConfig, adminConfig, err := loadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 先连接MySQL并创建数据库
	db, err := connectDB(dbConfig, "")
	if err != nil {
		fmt.Printf("连接MySQL失败: %v\n", err)
		os.Exit(1)
	}

	// 创建数据库
	createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbConfig.Database)
	if err := db.Exec(createDB).Error; err != nil {
		fmt.Printf("创建数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 3. 连接到目标数据库
	db, err = connectDB(dbConfig, dbConfig.Database)
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 4. 同步表结构
	models := []interface{}{&model.User{}, &model.Account{}, &model.Setting{}, &model.Monitor{}} // 添加 Monitor 模型
	fmt.Printf("开始迁移数据表...\n")

	for _, model := range models {
		// 获取表名
		stmt := &gorm.Statement{DB: db}
		stmt.Parse(model)
		tableName := stmt.Schema.Table

		// 获取当前表的列
		var columns []Column
		db.Raw(`
			SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
		`, tableName).Scan(&columns)

		// 记录当前列名
		currentColumns := make(map[string]bool)
		for _, col := range columns {
			if col.Name != "" {
				currentColumns[strings.ToLower(col.Name)] = true
			}
		}

		// 删除多余的列
		for colName := range currentColumns {
			// 跳过GORM默认字段
			if colName == "id" || strings.HasSuffix(colName, "_at") {
				continue
			}

			// 检查字段是否在模型中存在
			field := stmt.Schema.LookUpField(colName)
			if field == nil {
				sql := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", tableName, colName)
				if err := db.Exec(sql).Error; err != nil {
					fmt.Printf("删除列 %s 失败: %v\n", colName, err)
				} else {
					fmt.Printf("表 %s: 删除多余字段 %s\n", tableName, colName)
				}
			}
		}

		// 使用AutoMigrate添加或更新列
		if err := db.AutoMigrate(model); err != nil {
			fmt.Printf("同步表 %s 结构失败: %v\n", tableName, err)
			continue
		}
		fmt.Printf("表 %s: 迁移完成\n", tableName)
	}

	fmt.Println("所有数据表迁移完成")

	// 5. 创建管理员用户(如果启用)
	if *createAdmin {
		if err := createAdminUser(db, adminConfig); err != nil {
			fmt.Printf("创建管理员失败: %v\n", err)
		}
	}
}
