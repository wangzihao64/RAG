package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"rag/config"
	"rag/internal/model"
	"rag/internal/repository"
)

// 业务错误，handler 层据此映射成合适的响应
var (
	ErrDocumentTypeNotAllowed = errors.New("不支持的文件类型")
	ErrDocumentTooLarge       = errors.New("文件超过大小上限")
	ErrDocumentExists         = errors.New("该文件已存在于此知识库")
	ErrDocumentNotFound       = errors.New("文档不存在")
	ErrDocumentEmpty          = errors.New("文件为空")
)

// checkCollectionWritable 校验知识库存在且当前用户为 owner（有写权限）
func checkCollectionWritable(ctx context.Context, collectionID, userID uint) (*model.Collection, error) {
	col, err := repository.NewCollectionDao(ctx).FindByID(collectionID)
	if err != nil {
		return nil, err
	}
	if col == nil {
		return nil, ErrCollectionNotFound
	}
	if col.OwnerID != userID {
		return nil, ErrCollectionForbidden
	}
	return col, nil
}

// UploadDocument 上传文档到指定知识库：
// 校验权限/类型/大小 -> 计算 sha256 去重 -> 存盘 -> 建 pending 记录
func UploadDocument(ctx context.Context, collectionID, uploaderID uint, fileHeader *multipart.FileHeader) (*model.Document, error) {
	if _, err := checkCollectionWritable(ctx, collectionID, uploaderID); err != nil {
		return nil, err
	}

	if fileHeader.Size == 0 {
		return nil, ErrDocumentEmpty
	}
	if fileHeader.Size > config.MaxFileSizeMB*1024*1024 {
		return nil, ErrDocumentTooLarge
	}

	// 文件类型由扩展名判断
	fileType := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileHeader.Filename), "."))
	if !isAllowedType(fileType) {
		return nil, ErrDocumentTypeNotAllowed
	}

	// 计算内容 sha256，用于去重与命名
	hash, err := hashUploadedFile(fileHeader)
	if err != nil {
		return nil, err
	}

	docDao := repository.NewDocumentDao(ctx)
	existing, err := docDao.FindByCollectionAndHash(collectionID, hash)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDocumentExists
	}

	// 落盘：uploads/{collectionID}/{hash}.{ext}
	dir := filepath.Join(config.UploadDir, fmt.Sprintf("%d", collectionID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	dstPath := filepath.Join(dir, hash+"."+fileType)
	if err := saveUploadedFile(fileHeader, dstPath); err != nil {
		return nil, err
	}

	doc := &model.Document{
		CollectionID: collectionID,
		UploaderID:   uploaderID,
		Name:         fileHeader.Filename,
		FileType:     fileType,
		FilePath:     dstPath,
		FileSize:     fileHeader.Size,
		FileHash:     hash,
		Status:       model.DocStatusPending,
	}
	if err := docDao.CreateDocument(doc); err != nil {
		// 落库失败则回滚已写入的文件，避免留下孤儿文件
		_ = os.Remove(dstPath)
		return nil, err
	}
	return doc, nil
}

// ListDocuments 列出知识库下的文档：owner 可见，或该库为公开
func ListDocuments(ctx context.Context, collectionID, userID uint) ([]model.Document, error) {
	if _, err := GetCollection(ctx, collectionID, userID); err != nil {
		return nil, err
	}
	return repository.NewDocumentDao(ctx).ListByCollection(collectionID)
}

// GetDocument 获取文档详情，可见性跟随其所属知识库
func GetDocument(ctx context.Context, id, userID uint) (*model.Document, error) {
	doc, err := repository.NewDocumentDao(ctx).FindByID(id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, ErrDocumentNotFound
	}
	// 借用 collection 的可见性规则做鉴权
	if _, err := GetCollection(ctx, doc.CollectionID, userID); err != nil {
		return nil, err
	}
	return doc, nil
}

// DeleteDocument 删除文档：需对所属知识库有写权限，同时删除物理文件
func DeleteDocument(ctx context.Context, id, userID uint) error {
	docDao := repository.NewDocumentDao(ctx)
	doc, err := docDao.FindByID(id)
	if err != nil {
		return err
	}
	if doc == nil {
		return ErrDocumentNotFound
	}
	if _, err := checkCollectionWritable(ctx, doc.CollectionID, userID); err != nil {
		return err
	}
	if err := docDao.DeleteDocument(doc); err != nil {
		return err
	}
	// 记录已软删，物理文件尽力删除，失败不影响主流程
	_ = os.Remove(doc.FilePath)
	return nil
}

func isAllowedType(fileType string) bool {
	for _, t := range config.AllowedTypes {
		if t == fileType {
			return true
		}
	}
	return false
}

// hashUploadedFile 流式计算上传文件的 sha256
func hashUploadedFile(fileHeader *multipart.FileHeader) (string, error) {
	f, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// saveUploadedFile 将上传文件写入目标路径
func saveUploadedFile(fileHeader *multipart.FileHeader, dstPath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
