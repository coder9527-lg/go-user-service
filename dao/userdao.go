package dao

import (
	"errors"

	"github.com/coder9527-lg/go-user-service/model"
	"gorm.io/gorm"
)

// ErrUserNotFound 用户不存在。
var ErrUserNotFound = errors.New("user not found")

// ErrPhoneDuplicated 手机号已存在（并发注册时由唯一索引兜底）。
var ErrPhoneDuplicated = errors.New("phone already registered")

// UserRepo 用户数据访问对象，通过构造函数注入 *gorm.DB，
// 不再依赖全局单例，便于替换实现和并行测试。
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建 UserRepo。
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// GetByPhone 按手机号查询用户；查不到时返回 ErrUserNotFound。
func (r *UserRepo) GetByPhone(phone string) (*model.User, error) {
	user := &model.User{}
	// 用 First 而非 Find：First 在记录不存在时会返回 gorm.ErrRecordNotFound，
	// 调用方可以区分“用户不存在”和“零值对象”。
	err := r.db.Where("phone = ?", phone).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID 按主键查询用户；查不到时返回 ErrUserNotFound。
func (r *UserRepo) GetByID(id uint) (*model.User, error) {
	user := &model.User{}
	err := r.db.First(user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Create 创建用户；唯一索引冲突时返回 ErrPhoneDuplicated。
func (r *UserRepo) Create(user *model.User) error {
	err := r.db.Create(user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrPhoneDuplicated
	}
	return err
}
