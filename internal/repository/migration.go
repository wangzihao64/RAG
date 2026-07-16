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
}
