package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"rag/internal/service"
	"rag/pkg/response"
)

// chatRequest 是问答/检索接口的请求体
type chatRequest struct {
	Query string `json:"query" form:"query" binding:"required,min=1"`
	TopK  int    `json:"top_k" form:"top_k" binding:"omitempty,min=1,max=50"`
}

// Query 处理 POST /collections/:id/query —— 纯向量检索，返回命中片段（JSON）。
// 主要用于调试与前端“只看召回”场景。
func Query(c *gin.Context) {
	collectionID, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req chatRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "参数错误："+err.Error())
		return
	}

	userID := c.GetUint("user_id")
	chunks, err := service.Retrieve(c.Request.Context(), collectionID, userID, req.Query, req.TopK)
	if err != nil {
		documentErrorResponse(c, err)
		return
	}
	response.Success(c, chunks)
}

// Chat 处理 POST /collections/:id/chateval —— RAG 非流式问答。
func ChatEval(c *gin.Context) {
	collectionID, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req chatRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, "AI 回答生成失败："+err.Error())
		return
	}

	userID := c.GetUint("user_id")

	// 先检索：此时尚未写任何响应体，鉴权/参数类错误可走标准 JSON 响应。
	// 拆成“先检索、再流式生成”是为了让 sources 事件先于 message 下发。
	chunks, err := service.Retrieve(c.Request.Context(), collectionID, userID, req.Query, req.TopK)
	if err != nil {
		documentErrorResponse(c, err)
		return
	}
	messages := service.BuildRAGMessages(req.Query, chunks)
	resp, err := service.Answer(c.Request.Context(), messages)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "参数错误："+err.Error())
		return
	}
	var contexts []string
	if service.ShouldShowSources(resp) {
		contexts = make([]string, len(chunks))
		for i, chunk := range chunks {
			contexts[i] = chunk.Content
		}
	}
	response.Success(c, resp, contexts)
}

// Chat 处理 POST /collections/:id/chat —— RAG 流式问答，走 SSE。
// 事件序列：先 sources（引用来源），再若干 message（增量文本），最后 done；
// 流开始后若出错则发 error 事件。
func Chat(c *gin.Context) {
	collectionID, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req chatRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "参数错误："+err.Error())
		return
	}

	userID := c.GetUint("user_id")

	// 先检索：此时尚未写任何响应体，鉴权/参数类错误可走标准 JSON 响应。
	// 拆成“先检索、再流式生成”是为了让 sources 事件先于 message 下发。
	chunks, err := service.Retrieve(c.Request.Context(), collectionID, userID, req.Query, req.TopK)
	if err != nil {
		documentErrorResponse(c, err)
		return
	}

	// 检索成功，切换为 SSE 响应
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 关闭 Nginx 缓冲，保证逐段下发

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, 500, "当前服务器不支持流式响应")
		return
	}

	sendEvent := func(event string, data any) {
		c.SSEvent(event, data)
		flusher.Flush()
	}

	// 增量消息立即下发，完整答案仅用于生成结束后的来源判断。
	messages := service.BuildRAGMessages(req.Query, chunks)
	var answer strings.Builder
	if err := service.StreamAnswer(c.Request.Context(), messages, func(delta string) error {
		answer.WriteString(delta)
		sendEvent("message", delta)
		return nil
	}); err != nil {
		sendEvent("error", err.Error())
		return
	}

	if service.ShouldShowSources(answer.String()) {
		sendEvent("sources", chunks)
	}
	sendEvent("done", "")
}
