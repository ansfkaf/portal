// utils/s3/backup.go
package s3

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
)

// DBConfig 数据库配置
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// S3Config AWS S3配置
type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	BucketName      string
}

// BackupService 备份服务
type BackupService struct {
	DBConfig DBConfig
	S3Config S3Config
	Env      string // 环境：dev或prod
}

// NewBackupService 创建备份服务
func NewBackupService() *BackupService {
	return &BackupService{
		DBConfig: DBConfig{
			Host:     os.Getenv("MYSQL_HOST"),
			Port:     os.Getenv("MYSQL_PORT"),
			User:     os.Getenv("MYSQL_USERNAME"),
			Password: os.Getenv("MYSQL_PASSWORD"),
			Database: os.Getenv("MYSQL_DATABASE"),
		},
		S3Config: S3Config{
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			Region:          os.Getenv("AWS_DEFAULT_REGION"),
			BucketName:      os.Getenv("BUCKET_NAME"),
		},
		Env: os.Getenv("APP_ENV"),
	}
}

// BackupDatabase 备份数据库并上传到S3
func (s *BackupService) BackupDatabase(forceEnv string) (string, error) {
	// 1. 创建临时文件
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("%s_%s.sql", s.DBConfig.Database, timestamp)
	tempDir := os.TempDir()
	backupFilePath := filepath.Join(tempDir, backupFileName)

	// 打印配置信息（不要在生产环境启用，因为会泄露敏感信息）
	log.Printf("备份配置: DB=%s@%s:%s/%s, 输出文件=%s",
		s.DBConfig.User, s.DBConfig.Host, s.DBConfig.Port, s.DBConfig.Database, backupFilePath)

	// 2. 执行mysqldump命令
	cmd := exec.Command("mysqldump",
		"--host="+s.DBConfig.Host,
		"--port="+s.DBConfig.Port,
		"--user="+s.DBConfig.User,
		"--password="+s.DBConfig.Password,
		"--databases", s.DBConfig.Database,
		"--single-transaction",
		"--quick",
		"--lock-tables=false")

	// 创建输出文件
	outfile, err := os.Create(backupFilePath)
	if err != nil {
		return "", fmt.Errorf("创建备份文件失败: %v", err)
	}
	defer outfile.Close()

	// 设置命令输出
	var stderr bytes.Buffer
	cmd.Stdout = outfile
	cmd.Stderr = &stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mysqldump执行失败: %v, 错误信息: %s", err, stderr.String())
	}

	// 检查文件大小，确保备份文件不是空的
	fileInfo, err := os.Stat(backupFilePath)
	if err != nil {
		return "", fmt.Errorf("获取备份文件信息失败: %v", err)
	}

	if fileInfo.Size() == 0 {
		// 尝试直接向文件写入命令输出
		log.Println("备份文件为空，尝试使用管道方式执行...")

		// 关闭之前的文件
		outfile.Close()

		// 重新创建文件
		outfile, err = os.Create(backupFilePath)
		if err != nil {
			return "", fmt.Errorf("重新创建备份文件失败: %v", err)
		}
		defer outfile.Close()

		// 使用管道方式执行，可能更可靠
		cmd = exec.Command("mysqldump",
			"--host="+s.DBConfig.Host,
			"--port="+s.DBConfig.Port,
			"--user="+s.DBConfig.User,
			"--password="+s.DBConfig.Password,
			s.DBConfig.Database,
			"--single-transaction",
			"--quick",
			"--lock-tables=false")

		// 捕获标准输出和标准错误
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return "", fmt.Errorf("创建标准输出管道失败: %v", err)
		}

		stderr := bytes.Buffer{}
		cmd.Stderr = &stderr

		// 启动命令
		if err := cmd.Start(); err != nil {
			return "", fmt.Errorf("启动mysqldump命令失败: %v", err)
		}

		// 将输出复制到文件
		if _, err := io.Copy(outfile, stdout); err != nil {
			return "", fmt.Errorf("写入备份内容失败: %v", err)
		}

		// 等待命令完成
		if err := cmd.Wait(); err != nil {
			return "", fmt.Errorf("mysqldump命令执行失败: %v, 错误信息: %s", err, stderr.String())
		}

		// 再次检查文件大小
		fileInfo, err = os.Stat(backupFilePath)
		if err != nil {
			return "", fmt.Errorf("获取备份文件信息失败: %v", err)
		}

		if fileInfo.Size() == 0 {
			// 如果还是空的，尝试直接使用控制台命令创建
			log.Println("备份文件仍为空，尝试使用系统命令...")

			shellCmd := fmt.Sprintf("mysqldump --host=%s --port=%s --user=%s --password=%s %s > %s",
				s.DBConfig.Host,
				s.DBConfig.Port,
				s.DBConfig.User,
				s.DBConfig.Password,
				s.DBConfig.Database,
				backupFilePath)

			// 使用bash执行
			cmd = exec.Command("bash", "-c", shellCmd)
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("系统命令备份失败: %v, 错误信息: %s", err, stderr.String())
			}

			// 最后检查文件大小
			fileInfo, err = os.Stat(backupFilePath)
			if err != nil {
				return "", fmt.Errorf("获取备份文件信息失败: %v", err)
			}

			if fileInfo.Size() == 0 {
				return "", fmt.Errorf("尝试多种方法后备份文件仍为空")
			}
		}
	}

	log.Printf("备份文件 %s 大小: %d 字节", backupFilePath, fileInfo.Size())

	// 3. 确定S3目录路径
	s3Dir := "portal/"
	if forceEnv == "dev" || (forceEnv == "" && s.Env == "dev") {
		s3Dir = "portal/dev/"
	}

	// 4. 上传到S3
	s3Path, err := s.uploadToS3(backupFilePath, fmt.Sprintf("%s%s", s3Dir, backupFileName))
	if err != nil {
		return "", err
	}

	// 5. 清理临时文件
	os.Remove(backupFilePath)

	return s3Path, nil
}

// uploadToS3 上传文件到S3
func (s *BackupService) uploadToS3(filePath, s3Key string) (string, error) {
	// 创建AWS会话
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(s.S3Config.Region),
		Credentials: credentials.NewStaticCredentials(
			s.S3Config.AccessKeyID,
			s.S3Config.SecretAccessKey,
			"",
		),
	})
	if err != nil {
		return "", fmt.Errorf("创建AWS会话失败: %v", err)
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %v", err)
	}

	log.Printf("准备上传到S3: 文件=%s, 大小=%d bytes, S3键=%s", filePath, fileInfo.Size(), s3Key)

	// 创建S3客户端
	s3Client := s3.New(sess)

	// 上传到S3
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.S3Config.BucketName),
		Key:           aws.String(s3Key),
		Body:          file,
		ContentLength: aws.Int64(fileInfo.Size()),
		ContentType:   aws.String("application/sql"),
	})
	if err != nil {
		return "", fmt.Errorf("上传到S3失败: %v", err)
	}

	s3Path := fmt.Sprintf("s3://%s/%s", s.S3Config.BucketName, s3Key)
	log.Printf("成功上传到S3路径: %s", s3Path)
	return s3Path, nil
}

// 在RegisterBackupAPI函数中添加恢复数据库的路由
func RegisterBackupAPI(router *gin.Engine) {
	backupService := NewBackupService()

	// 现有的备份API
	router.POST("/admin/backup", func(c *gin.Context) {
		// 这里可以添加管理员身份验证
		s3Path, err := backupService.BackupDatabase("") // 使用当前环境配置
		if err != nil {
			log.Printf("备份失败: %v", err)
			c.JSON(500, gin.H{
				"success": false,
				"message": fmt.Sprintf("备份失败: %v", err),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "数据库备份成功",
			"data": gin.H{
				"s3Path": s3Path,
				"time":   time.Now().Format(time.RFC3339),
			},
		})
	})

	// 新增的恢复数据库API
	router.POST("/admin/restore", func(c *gin.Context) {
		// 这里可以添加管理员身份验证
		var request RestoreDatabaseRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": fmt.Sprintf("请求参数错误: %v", err),
			})
			return
		}

		// 执行数据库恢复
		if err := backupService.RestoreDatabase(request.BackupFilePath); err != nil {
			log.Printf("恢复失败: %v", err)
			c.JSON(500, gin.H{
				"success": false,
				"message": fmt.Sprintf("恢复失败: %v", err),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "数据库恢复成功",
			"data": gin.H{
				"backupFilePath": request.BackupFilePath,
				"time":           time.Now().Format(time.RFC3339),
			},
		})
	})
}

// ScheduleBackupTask 定时备份任务
func ScheduleBackupTask() {
	backupService := NewBackupService()

	// 仅在生产环境启用自动备份
	if backupService.Env != "prod" {
		fmt.Println("当前为开发环境，自动备份功能已禁用")
		return
	}

	// 每天凌晨2点执行备份
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
			duration := next.Sub(now)

			fmt.Printf("下一次自动备份将在 %s 进行\n", next.Format("2006-01-02 15:04:05"))
			time.Sleep(duration)

			s3Path, err := backupService.BackupDatabase("") // 使用当前环境配置
			if err != nil {
				fmt.Printf("定时备份失败: %v\n", err)
			} else {
				fmt.Printf("定时备份成功, 路径: %s\n", s3Path)
			}
		}
	}()
}

// InitBackup 初始化备份功能
func InitBackup(router *gin.Engine) {
	// 注册API
	RegisterBackupAPI(router)

	// 启动定时备份
	ScheduleBackupTask()
}

// RestoreDatabaseRequest 恢复数据库请求
type RestoreDatabaseRequest struct {
	BackupFilePath string `json:"backupFilePath" binding:"required"`
}

// RestoreDatabase 从备份文件恢复数据库
func (s *BackupService) RestoreDatabase(backupFilePath string) error {
	// 验证文件是否存在
	if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupFilePath)
	}

	log.Printf("开始从文件 %s 恢复数据库 %s", backupFilePath, s.DBConfig.Database)

	// 构建mysql命令参数
	args := []string{
		"-h" + s.DBConfig.Host,
		"-P" + s.DBConfig.Port,
		"-u" + s.DBConfig.User,
		"-p" + s.DBConfig.Password,
		s.DBConfig.Database,
	}

	// 创建命令
	cmd := exec.Command("mysql", args...)

	// 打开备份文件作为输入
	backupFile, err := os.Open(backupFilePath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer backupFile.Close()

	// 设置文件作为标准输入
	cmd.Stdin = backupFile

	// 捕获标准错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("恢复数据库失败: %v, 错误信息: %s", err, stderr.String())
	}

	log.Printf("成功从 %s 恢复数据库 %s", backupFilePath, s.DBConfig.Database)
	return nil
}
