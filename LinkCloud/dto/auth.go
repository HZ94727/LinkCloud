package dto

type SendCaptchaRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
	UserName string `json:"user_name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	ID             uint64 `json:"id"`
	UserName       string `json:"user_name"`
	Email          string `json:"email"`
	Quota          uint32 `json:"quota"`
	UsedQuota      uint32 `json:"used_quota"`
	RemainingQuota uint32 `json:"remaining_quota"`
	Token          string `json:"token"`
	CreatedAt      int64  `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	UserName string `json:"user_name" binding:"required"`
	Password string `json:"password" binding:"required"`
	Captcha  string `json:"captcha" binding:"required,len=6"`
}

type RegisterResponse struct {
	ID             uint64 `json:"id"`
	Email          string `json:"email"`
	UserName       string `json:"user_name"`
	CreatedAt      int64  `json:"created_at"`
	Quota          uint32 `json:"quota"`
	UsedQuota      uint32 `json:"used_quota"`
	RemainingQuota uint32 `json:"remaining_quota"`
}
