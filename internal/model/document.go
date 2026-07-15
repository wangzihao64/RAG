package model

import (
	"gorm.io/gorm"
)

// 文档处理状态：上传后需要经过解析、切分、向量化才能被检索
const (
	DocStatusPending    = "pending"    // 已上传，待处理
	DocStatusProcessing = "processing" // 解析/向量化中
	DocStatusReady      = "ready"      // 可检索
	DocStatusFailed     = "failed"     // 处理失败
)

type Document struct {
	*gorm.Model
	CollectionID uint `gorm:"not null;index" json:"collection_id"`
	UploaderID   uint `gorm:"not null;index" json:"uploader_id"`
	// 文档名（一般是原始文件名）
	Name string `gorm:"type:varchar(256);not null" json:"name"`
	// 文件类型：pdf / md / txt / docx …，决定用哪种解析器
	FileType string `gorm:"type:varchar(32);not null" json:"file_type"` // pdf / md / txt / docx ...
	// 原始文件的存储路径（本地磁盘或对象存储的 key）
	FilePath string `gorm:"type:varchar(512);not null" json:"file_path"`
	// 文件大小，单位字节
	FileSize int64 `gorm:"not null;default:0" json:"file_size"` // 字节
	// 文件内容的 sha256，用来做去重（同一内容不重复处理）
	FileHash string `gorm:"type:varchar(64);index" json:"file_hash"` // sha256，用于去重
	// 处理状态机：pending→processing→ready/failed，带索引方便筛"待处理"
	Status string `gorm:"type:varchar(16);not null;default:pending;index" json:"status"`
	// 当 Status=failed 时，记录失败原因；成功时为空（omitempty 不输出）
	ErrorMsg string `gorm:"type:text" json:"error_msg,omitempty"` // status=failed 时的失败原因
	// 切分后产生多少文本块，也就是写入 Milvus 的向量条数
	ChunkCount int `gorm:"not null;default:0" json:"chunk_count"`

	Collection Collection `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
	Uploader   User       `gorm:"foreignKey:UploaderID" json:"uploader,omitempty"`
}

func (Document) TableName() string {
	return "documents"
}
