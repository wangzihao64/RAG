package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"rag/pkg/jwt"
	"rag/pkg/response"
)

// JWTAuth 校验 Authorization 头中的 Bearer token，
// 通过后把用户信息写入 gin.Context 供后续 handler 使用
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Fail(c, http.StatusUnauthorized, 401, "未携带认证信息")
			c.Abort()
			return
		}

		// 期望格式：Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Fail(c, http.StatusUnauthorized, 401, "认证格式错误")
			c.Abort()
			return
		}

		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, 401, "无效或已过期的 token")
			c.Abort()
			return
		}

		// 下游 handler 通过 c.GetUint("user_id") 等取用
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}
