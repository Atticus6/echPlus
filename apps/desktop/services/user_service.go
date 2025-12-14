package services

import (
	"github.com/atticus6/echPlus/apps/desktop/database"
	"github.com/atticus6/echPlus/apps/desktop/models"
)

type UserService struct{}

// Create 创建用户
func (s *UserService) Create(name, email string) (*models.User, error) {
	user := &models.User{
		Name:  name,
		Email: email,
	}
	if err := database.GetDB().Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID 根据ID获取用户
func (s *UserService) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := database.GetDB().First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAll 获取所有用户
func (s *UserService) GetAll() ([]models.User, error) {
	var users []models.User
	if err := database.GetDB().Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Update 更新用户
func (s *UserService) Update(id uint, name, email string) (*models.User, error) {
	var user models.User
	if err := database.GetDB().First(&user, id).Error; err != nil {
		return nil, err
	}
	user.Name = name
	user.Email = email
	if err := database.GetDB().Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Delete 删除用户
func (s *UserService) Delete(id uint) error {
	return database.GetDB().Delete(&models.User{}, id).Error
}
