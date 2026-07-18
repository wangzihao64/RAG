package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"rag/internal/model"
	"rag/internal/service"
	"rag/pkg/response"
)

// collectionView 是返回给前端的知识库信息
type collectionView struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     uint   `json:"owner_id"`
	IsPublic    bool   `json:"is_public"`
}

func toCollectionView(c *model.Collection) collectionView {
	return collectionView{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		OwnerID:     c.OwnerID,
		IsPublic:    c.IsPublic,
	}
}

// parseIDParam 解析路径参数 :id
func parseIDParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "非法的 id")
		return 0, false
	}
	return uint(id), true
}

// collectionErrorResponse 把 service 层业务错误映射成 HTTP 响应
func collectionErrorResponse(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCollectionNameTaken):
		response.Fail(c, http.StatusConflict, 409, err.Error())
	case errors.Is(err, service.ErrCollectionNotFound):
		response.Fail(c, http.StatusNotFound, 404, err.Error())
	case errors.Is(err, service.ErrCollectionForbidden):
		response.Fail(c, http.StatusForbidden, 403, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, 500, "服务器内部错误")
	}
}

// CollectionCreate 处理 POST /collections
func CollectionCreate(c *gin.Context) {
	var req service.CreateCollectionRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	userID := c.GetUint("user_id")
	col, err := req.Create(c.Request.Context(), userID)
	if err != nil {
		collectionErrorResponse(c, err)
		return
	}
	response.Success(c, toCollectionView(col))
}

// CollectionList 处理 GET /collections，列出当前用户的知识库
func CollectionList(c *gin.Context) {
	userID := c.GetUint("user_id")
	list, err := service.ListCollections(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, "服务器内部错误")
		return
	}

	views := make([]collectionView, 0, len(list))
	for i := range list {
		views = append(views, toCollectionView(&list[i]))
	}
	response.Success(c, views)
}

// CollectionDetail 处理 GET /collections/:id
func CollectionDetail(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	col, err := service.GetCollection(c.Request.Context(), id, userID)
	if err != nil {
		collectionErrorResponse(c, err)
		return
	}
	response.Success(c, toCollectionView(col))
}

// CollectionUpdate 处理 PUT /collections/:id
func CollectionUpdate(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req service.UpdateCollectionRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	userID := c.GetUint("user_id")
	col, err := req.Update(c.Request.Context(), id, userID)
	if err != nil {
		collectionErrorResponse(c, err)
		return
	}
	response.Success(c, toCollectionView(col))
}

// CollectionDelete 处理 DELETE /collections/:id
func CollectionDelete(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	if err := service.DeleteCollection(c.Request.Context(), id, userID); err != nil {
		collectionErrorResponse(c, err)
		return
	}
	response.Success(c, nil)
}
