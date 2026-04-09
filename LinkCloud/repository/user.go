package repository

import (
	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/model"
	"gorm.io/gorm"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUserName(userName string) (*model.User, error) {
	var user model.User
	if err := database.DB.Where("user_name = ?", userName).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *model.User) error {
	return database.DB.Create(user).Error
}

func (r *UserRepository) GetByID(id uint64) (*model.User, error) {
	var user model.User
	if err := database.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) IncreaseUsedQuota(db *gorm.DB, id uint64, delta uint32) error {
	if db == nil {
		db = database.DB
	}

	return db.Model(&model.User{}).
		Where("id = ?", id).
		Update("used_quota", gorm.Expr("used_quota + ?", delta)).Error
}

func (r *UserRepository) GetByUserNameExceptID(userName string, excludedID uint64) (*model.User, error) {
	var user model.User
	if err := database.DB.Where("user_name = ? AND id != ?", userName, excludedID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(db *gorm.DB, user *model.User, updates map[string]any) error {
	if db == nil {
		db = database.DB
	}

	return db.Model(user).Updates(updates).Error
}
