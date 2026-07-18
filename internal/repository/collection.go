package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"rag/internal/model"
)

// CollectionDao 封装 collections 表的数据访问
type CollectionDao struct {
	*gorm.DB
}

func NewCollectionDao(ctx context.Context) *CollectionDao {
	return &CollectionDao{NewDBClient(ctx)}
}

// CreateCollection 插入一条知识库记录
func (d *CollectionDao) CreateCollection(c *model.Collection) error {
	return d.DB.Create(c).Error
}

// FindByID 按主键查询，未找到返回 (nil, nil)
func (d *CollectionDao) FindByID(id uint) (*model.Collection, error) {
	var c model.Collection
	err := d.First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// FindByOwnerAndName 查同一 owner 下的同名知识库，用于查重，未找到返回 (nil, nil)
func (d *CollectionDao) FindByOwnerAndName(ownerID uint, name string) (*model.Collection, error) {
	var c model.Collection
	err := d.Where("owner_id = ? AND name = ?", ownerID, name).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListByOwner 列出某用户拥有的全部知识库
func (d *CollectionDao) ListByOwner(ownerID uint) ([]model.Collection, error) {
	var list []model.Collection
	err := d.Where("owner_id = ?", ownerID).Order("id DESC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateCollection 保存对知识库的修改
func (d *CollectionDao) UpdateCollection(c *model.Collection) error {
	return d.DB.Save(c).Error
}

// DeleteCollection 软删除知识库
func (d *CollectionDao) DeleteCollection(c *model.Collection) error {
	return d.DB.Delete(c).Error
}
