package router

import (
	"net/http"
	"rag/internal/middleware"

	"github.com/gin-gonic/gin"

	"rag/config"
)

// NewRouter 构建 Gin 引擎并注册路由
func NewRouter() *gin.Engine {
	// 根据配置切换 Gin 运行模式，release 模式下关闭调试日志
	if config.AppModel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.Cors())
	v1 := r.Group("/api/v1")
	{
		// 健康检查：确认服务存活、可连库后续再扩展
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}
	return r
}
