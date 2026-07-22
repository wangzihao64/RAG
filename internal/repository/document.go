package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"rag/internal/model"
)

// DocumentDao 封装 documents 表的数据访问
type DocumentDao struct {
	*gorm.DB
}

func NewDocumentDao(ctx context.Context) *DocumentDao {
	return &DocumentDao{NewDBClient(ctx)}
}

// CreateDocument 插入一条文档记录
func (d *DocumentDao) CreateDocument(doc *model.Document) error {
	return d.DB.Create(doc).Error
}

// FindByID 按主键查询，未找到返回 (nil, nil)
func (d *DocumentDao) FindByID(id uint) (*model.Document, error) {
	var doc model.Document
	err := d.First(&doc, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// FindByCollectionAndHash 查同一知识库下相同内容的文档，用于去重，未找到返回 (nil, nil)
func (d *DocumentDao) FindByCollectionAndHash(collectionID uint, hash string) (*model.Document, error) {
	var doc model.Document
	err := d.Where("collection_id = ? AND file_hash = ?", collectionID, hash).First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// ListByCollection 列出某知识库下的全部文档
func (d *DocumentDao) ListByCollection(collectionID uint) ([]model.Document, error) {
	var list []model.Document
	err := d.Where("collection_id = ?", collectionID).Order("id DESC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteDocument 软删除文档
func (d *DocumentDao) DeleteDocument(doc *model.Document) error {
	return d.DB.Delete(doc).Error
}
