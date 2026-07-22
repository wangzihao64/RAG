package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"rag/internal/model"
	"rag/internal/service"
	"rag/pkg/response"
)

// documentView 是返回给前端的文档信息
type documentView struct {
	ID           uint   `json:"id"`
	CollectionID uint   `json:"collection_id"`
	UploaderID   uint   `json:"uploader_id"`
	Name         string `json:"name"`
	FileType     string `json:"file_type"`
	FileSize     int64  `json:"file_size"`
	FileHash     string `json:"file_hash"`
	Status       string `json:"status"`
	ErrorMsg     string `json:"error_msg,omitempty"`
	ChunkCount   int    `json:"chunk_count"`
}

func toDocumentView(d *model.Document) documentView {
	return documentView{
		ID:           d.ID,
		CollectionID: d.CollectionID,
		UploaderID:   d.UploaderID,
		Name:         d.Name,
		FileType:     d.FileType,
		FileSize:     d.FileSize,
		FileHash:     d.FileHash,
		Status:       d.Status,
		ErrorMsg:     d.ErrorMsg,
		ChunkCount:   d.ChunkCount,
	}
}

// documentErrorResponse 把 service 层业务错误映射成 HTTP 响应
func documentErrorResponse(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDocumentExists):
		response.Fail(c, http.StatusConflict, 409, err.Error())
	case errors.Is(err, service.ErrDocumentNotFound), errors.Is(err, service.ErrCollectionNotFound):
		response.Fail(c, http.StatusNotFound, 404, err.Error())
	case errors.Is(err, service.ErrCollectionForbidden):
		response.Fail(c, http.StatusForbidden, 403, err.Error())
	case errors.Is(err, service.ErrDocumentTypeNotAllowed),
		errors.Is(err, service.ErrDocumentTooLarge),
		errors.Is(err, service.ErrDocumentEmpty):
		response.Fail(c, http.StatusBadRequest, 400, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, 500, "服务器内部错误")
	}
}

// DocumentUpload 处理 POST /collections/:id/documents，multipart 上传
func DocumentUpload(c *gin.Context) {
	collectionID, ok := parseIDParam(c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "缺少上传文件 file")
		return
	}

	userID := c.GetUint("user_id")
	doc, err := service.UploadDocument(c.Request.Context(), collectionID, userID, fileHeader)
	if err != nil {
		documentErrorResponse(c, err)
		return
	}
	response.Success(c, toDocumentView(doc))
}

// DocumentList 处理 GET /collections/:id/documents
func DocumentList(c *gin.Context) {
	collectionID, ok := parseIDParam(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	list, err := service.ListDocuments(c.Request.Context(), collectionID, userID)
	if err != nil {
		documentErrorResponse(c, err)
		return
	}

	views := make([]documentView, 0, len(list))
	for i := range list {
		views = append(views, toDocumentView(&list[i]))
	}
	response.Success(c, views)
}

// DocumentDetail 处理 GET /documents/:id
func DocumentDetail(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	doc, err := service.GetDocument(c.Request.Context(), id, userID)
	if err != nil {
		documentErrorResponse(c, err)
		return
	}
	response.Success(c, toDocumentView(doc))
}

// DocumentDelete 处理 DELETE /documents/:id
func DocumentDelete(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	if err := service.DeleteDocument(c.Request.Context(), id, userID); err != nil {
		documentErrorResponse(c, err)
		return
	}
	response.Success(c, nil)
}
