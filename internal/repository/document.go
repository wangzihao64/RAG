package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// ListPendingDocuments() 列出status为pending状态的全部文档
func (d *DocumentDao) ListPendingDocuments() ([]model.Document, error) {
	var docs []model.Document
	err := d.Where("status = ?", model.DocStatusPending).Find(&docs).Error
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// ClaimPendingDocuments 原子领取最多 limit 条待处理文档。
// 行锁在事务提交前将领取到的记录更新为 processing，因此多个 worker
// 或多个服务实例并发轮询时，同一文档只会被其中一个领取。
func (d *DocumentDao) ClaimPendingDocuments(limit int) ([]model.Document, error) {
	if limit <= 0 {
		return nil, nil
	}

	var docs []model.Document
	err := d.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", model.DocStatusPending).
			Order("id ASC").
			Limit(limit).
			Find(&docs).Error; err != nil {
			return err
		}
		if len(docs) == 0 {
			return nil
		}

		ids := make([]uint, len(docs))
		for i, doc := range docs {
			ids[i] = doc.ID
		}
		return tx.Model(&model.Document{}).
			Where("id IN ? AND status = ?", ids, model.DocStatusPending).
			Updates(map[string]any{
				"status":    model.DocStatusProcessing,
				"error_msg": "",
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// updateStatus 更新文档状态与错误信息的函数。
func (d *DocumentDao) UpdateStatus(docID uint, status string, errMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errMsg != "" {
		updates["error_msg"] = errMsg
	} else {
		updates["error_msg"] = "" // 清空之前的错误
	}
	if err := d.DB.Model(&model.Document{}).Where("id = ?", docID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}
