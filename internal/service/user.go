package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"rag/internal/model"
	"rag/internal/repository"
	"rag/pkg/jwt"
)

// 业务错误，handler 层据此映射成合适的响应
var (
	ErrUsernameTaken      = errors.New("用户名已被占用")
	ErrEmailTaken         = errors.New("邮箱已被注册")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("账号已被禁用")
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 注册新用户：查重 -> bcrypt 加密 -> 落库
func (service *RegisterRequest) Register(ctx context.Context) (*model.User, error) {
	userDao := repository.NewUserDao(ctx)
	existing, err := userDao.FindByUsername(service.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	existing, err = userDao.FindByEmail(service.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(service.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     service.Username,
		Email:        service.Email,
		PasswordHash: string(hash),
		Role:         model.RoleUser,
		Status:       model.UserStatusActive,
	}
	if err := userDao.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login 校验用户名密码，成功则签发 JWT
func (service *LoginRequest) Login(ctx context.Context) (string, *model.User, error) {
	userDao := repository.NewUserDao(ctx)
	user, err := userDao.FindByUsername(service.Username)
	if err != nil {
		return "", nil, err
	}
	// 用户不存在也走密码比对失败分支，避免暴露"用户是否存在"
	if user == nil {
		return "", nil, ErrInvalidCredentials
	}
	if user.Status == model.UserStatusDisabled {
		return "", nil, ErrUserDisabled
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(service.Password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := jwt.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// GetUserByID 按 ID 查询用户
func GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	return repository.NewUserDao(ctx).FindByID(id)
}
