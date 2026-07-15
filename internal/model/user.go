package model

import (
	"gorm.io/gorm"
)

// 用户角色
const (
	RoleAdmin = "admin" // 系统管理员
	RoleUser  = "user"  // 普通用户
)

// 用户状态
const (
	UserStatusActive   = "active"   // 正常
	UserStatusDisabled = "disabled" // 禁用
)

type User struct {
	*gorm.Model
	Username     string `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Email        string `gorm:"type:varchar(128);uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"type:varchar(256);not null" json:"-"`
	Role         string `gorm:"type:varchar(16);not null;default:user" json:"role"`
	Status       string `gorm:"type:varchar(16);not null;default:active" json:"status"`

	Collections []Collection `gorm:"foreignKey:OwnerID" json:"collections,omitempty"`
}

func (User) TableName() string {
	return "users"
}
