package model

import (
	"gorm.io/gorm"
)

// 权限级别，控制用户对非自有 collection 的访问
const (
	PermRead  = "read"  // 只读：可检索、查看文档
	PermWrite = "write" // 读写：可上传、删除文档
	PermAdmin = "admin" // 管理：可修改 collection、授权他人
)

// Permission 授予某用户对某 collection 的访问权限（owner 天然拥有全部权限，无需记录）
type Permission struct {
	*gorm.Model
	UserID       uint   `gorm:"not null;uniqueIndex:idx_user_collection" json:"user_id"`
	CollectionID uint   `gorm:"not null;index;uniqueIndex:idx_user_collection" json:"collection_id"`
	Level        string `gorm:"type:varchar(16);not null;default:read" json:"level"`
	GrantedBy    uint   `gorm:"not null" json:"granted_by"`

	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Collection Collection `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
}

func (Permission) TableName() string {
	return "permissions"
}
