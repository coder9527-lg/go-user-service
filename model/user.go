package model

import "gorm.io/gorm"

// User 用户表模型。
type User struct {
	gorm.Model
	Name     string `gorm:"type:varchar(25);not null" json:"name"`
	Password string `gorm:"type:varchar(100);not null" json:"-"` // 密码哈希，绝不下发给前端
	Phone    string `gorm:"size:20;not null;uniqueIndex" json:"phone"`
}
