package model

import (
	"gorm.io/gorm"
)

// Collection 是文档的逻辑分组（知识库），RAG 检索以 collection 为范围
type Collection struct {
	*gorm.Model
	//知识库名称
	Name string `gorm:"type:varchar(128);not null;uniqueIndex:idx_owner_name" json:"name"`
	//描述
	Description string `gorm:"type:text" json:"description"`
	//谁创建的/负责人
	OwnerID  uint `gorm:"not null;index;uniqueIndex:idx_owner_name" json:"owner_id"`
	IsPublic bool `gorm:"not null;default:false" json:"is_public"`

	Owner     User       `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Documents []Document `gorm:"foreignKey:CollectionID" json:"documents,omitempty"`
}

func (Collection) TableName() string {
	return "collections"
}
