package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一的 JSON 响应结构
type Response struct {
	Code int    `json:"code"` // 业务状态码，0 表示成功
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// Success 返回成功响应
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// Fail 返回失败响应，httpStatus 为 HTTP 状态码，code 为业务码
func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
	})
}
