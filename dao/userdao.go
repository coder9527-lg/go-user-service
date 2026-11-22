package dao

import (
	"errors"

	"github.com/coder9527-lg/go-user-service/db"
	"github.com/coder9527-lg/go-user-service/model"
	"gorm.io/gorm"
)

// ErrUserNotFound 用户不存在。
var ErrUserNotFound = errors.New("user not found")

// ErrPhoneDuplicated 手机号已存在（并发注册时由唯一索引兜底）。
var ErrPhoneDuplicated = errors.New("phone already registered")

// GetUserByPhone 按手机号查询用户；查不到时返回 ErrUserNotFound。
func GetUserByPhone(phone string) (*model.User, error) {
	user := &model.User{}
	// 用 First 而非 Find：First 在记录不存在时会返回 gorm.ErrRecordNotFound，
	// 原实现用 Find 查不到也不报错，调用方无法区分“用户不存在”和“零值对象”。
	err := db.DB.Where("phone = ?", phone).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID 按主键查询用户；查不到时返回 ErrUserNotFound。
func GetUserByID(id uint) (*model.User, error) {
	user := &model.User{}
	err := db.DB.First(user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser 创建用户；唯一索引冲突时返回 ErrPhoneDuplicated。
func CreateUser(user *model.User) error {
	err := db.DB.Create(user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrPhoneDuplicated
	}
	return err
}
