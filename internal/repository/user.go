package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"rag/internal/model"
)

// UserRepository 封装 users 表的数据访问
type UserDao struct {
	*gorm.DB
}

func NewUserDao(ctx context.Context) *UserDao {
	return &UserDao{NewDBClient(ctx)}
}

// Create 插入一条用户记录
func (u *UserDao) CreateUser(user *model.User) error {
	return u.DB.Create(user).Error
}

// FindByUsername 按用户名查询，未找到返回 (nil, nil)
func (u *UserDao) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := u.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail 按邮箱查询，未找到返回 (nil, nil)
func (u *UserDao) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := u.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 按主键查询，未找到返回 (nil, nil)
func (u *UserDao) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := u.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
