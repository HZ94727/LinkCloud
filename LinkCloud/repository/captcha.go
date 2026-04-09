package repository

import (
	"fmt"
	"time"

	"gitea.com/hz/linkcloud/database"
)

type CaptchaRepository struct{}

func NewCaptchaRepository() *CaptchaRepository {
	return &CaptchaRepository{}
}

func (r *CaptchaRepository) Get(email string) (string, error) {
	codeKey := fmt.Sprintf("captcha:%s", email)
	return database.Redis.Get(database.Ctx, codeKey).Result()
}

func (r *CaptchaRepository) Set(email, code string, expiration time.Duration) error {
	codeKey := fmt.Sprintf("captcha:%s", email)
	return database.Redis.Set(database.Ctx, codeKey, code, expiration).Err()
}

func (r *CaptchaRepository) Delete(email string) error {
	codeKey := fmt.Sprintf("captcha:%s", email)
	return database.Redis.Del(database.Ctx, codeKey).Err()
}
