package repository

import (
	"fmt"
	"rag/internal/model"
)

func Migration() {
	if err := DB.AutoMigrate(
		&model.User{},
		&model.Collection{},
		&model.Document{},
		&model.Permission{},
	); err != nil {
		panic(fmt.Sprintf("数据库迁移失败: %v", err))
	}
	if err := ensureCollectionNameIndex(); err != nil {
		panic(fmt.Sprintf("知识库名称索引迁移失败: %v", err))
	}
}

// ensureCollectionNameIndex 只约束未软删除的知识库名称，允许复用已删除知识库的名称。
func ensureCollectionNameIndex() error {
	if err := DB.Exec("DROP INDEX IF EXISTS idx_owner_name").Error; err != nil {
		return err
	}
	return DB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_owner_name
		ON collections (owner_id, name)
		WHERE deleted_at IS NULL
	`).Error
}
