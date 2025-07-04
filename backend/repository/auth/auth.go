// repository/auth/auth.go
package auth

import (
	"errors"
	"portal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{
		db: db,
	}
}

// ValidateCredentials 验证用户凭证
func (r *AuthRepository) ValidateCredentials(email, password string) (*model.User, error) {
	var user model.User

	// 查询用户
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, result.Error
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("密码错误")
	}

	return &user, nil
}

// CreateUser 创建新用户
func (r *AuthRepository) CreateUser(email, password string) (*model.User, error) {
	// 检查邮箱是否已存在
	var existingUser model.User
	if result := r.db.Where("email = ?", email).First(&existingUser); result.Error == nil {
		return nil, errors.New("该邮箱已被注册")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// 创建新用户
	user := &model.User{
		Email:    email,
		Password: password,
		IsAdmin:  0, // 默认非管理员
	}

	// 验证邮箱格式
	if err := user.ValidateEmail(); err != nil {
		return nil, err
	}

	// 创建用户 - 密码验证和加密会在 model.User 的 BeforeCreate 钩子中自动处理
	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}
