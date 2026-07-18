package service

import (
	"context"
	"errors"

	"rag/internal/model"
	"rag/internal/repository"
)

// 业务错误，handler 层据此映射成合适的响应
var (
	ErrCollectionNameTaken = errors.New("同名知识库已存在")
	ErrCollectionNotFound  = errors.New("知识库不存在")
	ErrCollectionForbidden = errors.New("无权操作该知识库")
)

type CreateCollectionRequest struct {
	Name        string `json:"name" form:"name" binding:"required,min=1,max=128"`
	Description string `json:"description" form:"description" binding:"max=1000"`
	IsPublic    bool   `json:"is_public" form:"is_public"`
}

type UpdateCollectionRequest struct {
	Name        string `json:"name" form:"name" binding:"required,min=1,max=128"`
	Description string `json:"description" form:"description" binding:"max=1000"`
	IsPublic    bool   `json:"is_public" form:"is_public"`
}

// Create 创建知识库：同一 owner 下不能重名
func (service *CreateCollectionRequest) Create(ctx context.Context, ownerID uint) (*model.Collection, error) {
	dao := repository.NewCollectionDao(ctx)

	existing, err := dao.FindByOwnerAndName(ownerID, service.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrCollectionNameTaken
	}

	c := &model.Collection{
		Name:        service.Name,
		Description: service.Description,
		OwnerID:     ownerID,
		IsPublic:    service.IsPublic,
	}
	if err := dao.CreateCollection(c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListCollections 列出当前用户拥有的知识库
func ListCollections(ctx context.Context, ownerID uint) ([]model.Collection, error) {
	return repository.NewCollectionDao(ctx).ListByOwner(ownerID)
}

// GetCollection 获取知识库详情：owner 可见，或该库为公开
func GetCollection(ctx context.Context, id, userID uint) (*model.Collection, error) {
	c, err := repository.NewCollectionDao(ctx).FindByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCollectionNotFound
	}
	if c.OwnerID != userID && !c.IsPublic {
		return nil, ErrCollectionForbidden
	}
	return c, nil
}

// Update 修改知识库：仅 owner 可改，改名时仍需查重
func (service *UpdateCollectionRequest) Update(ctx context.Context, id, userID uint) (*model.Collection, error) {
	dao := repository.NewCollectionDao(ctx)

	c, err := dao.FindByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCollectionNotFound
	}
	if c.OwnerID != userID {
		return nil, ErrCollectionForbidden
	}

	// 改了名字才需要查重，且排除自己
	if service.Name != c.Name {
		existing, err := dao.FindByOwnerAndName(userID, service.Name)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrCollectionNameTaken
		}
	}

	c.Name = service.Name
	c.Description = service.Description
	c.IsPublic = service.IsPublic
	if err := dao.UpdateCollection(c); err != nil {
		return nil, err
	}
	return c, nil
}

// DeleteCollection 删除知识库：仅 owner 可删
func DeleteCollection(ctx context.Context, id, userID uint) error {
	dao := repository.NewCollectionDao(ctx)

	c, err := dao.FindByID(id)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrCollectionNotFound
	}
	if c.OwnerID != userID {
		return ErrCollectionForbidden
	}
	return dao.DeleteCollection(c)
}
