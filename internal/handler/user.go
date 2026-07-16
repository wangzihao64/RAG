package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"rag/internal/model"
	"rag/internal/service"
	"rag/pkg/response"
)

// userView 是返回给前端的用户信息（不含敏感字段）
type userView struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

func toUserView(u *model.User) userView {
	return userView{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		Role:     u.Role,
		Status:   u.Status,
	}
}

// Register 处理 POST /auth/register
func UserRegister(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	user, err := req.Register(c.Request.Context())
	if err != nil {
		if errors.Is(err, service.ErrUsernameTaken) || errors.Is(err, service.ErrEmailTaken) {
			response.Fail(c, http.StatusConflict, 409, err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, 500, "注册失败")
		return
	}
	response.Success(c, toUserView(user))
}

// Login 处理 POST /auth/login
func UserLogin(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	token, user, err := req.Login(c.Request.Context())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Fail(c, http.StatusUnauthorized, 401, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserDisabled) {
			response.Fail(c, http.StatusForbidden, 403, err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, 500, "登录失败")
		return
	}
	response.Success(c, gin.H{
		"token": token,
		"user":  toUserView(user),
	})
}

// Profile 处理 GET /user/profile，返回当前登录用户信息（需鉴权）
func Profile(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := service.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		response.Fail(c, http.StatusNotFound, 404, "用户不存在")
		return
	}
	response.Success(c, toUserView(user))
}
