package router

import (
	"net/http"
	"rag/internal/handler"
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

		// 认证相关：注册、登录，无需鉴权
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.UserRegister)
			auth.POST("/login", handler.UserLogin)
		}

		// 需要登录的接口
		user := v1.Group("/user")
		user.Use(middleware.JWTAuth())
		{
			user.GET("/profile", handler.Profile)
		}

		// 知识库管理，全部需要登录
		collections := v1.Group("/collections")
		collections.Use(middleware.JWTAuth())
		{
			collections.POST("", handler.CollectionCreate)
			collections.GET("", handler.CollectionList)
			collections.GET("/:id", handler.CollectionDetail)
			collections.PUT("/:id", handler.CollectionUpdate)
			collections.DELETE("/:id", handler.CollectionDelete)
		}
	}
	return r
}
