package dto

type UserInfoResponse struct {
	ID             uint64 `json:"id"`
	UserName       string `json:"user_name"`
	Email          string `json:"email"`
	UsedQuota      uint32 `json:"used_quota"`
	Quota          uint32 `json:"quota"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	RemainingQuota uint32 `json:"remaining_quota"`
}

type UpdateUserInfoRequest struct {
	UserName    *string `json:"user_name"`
	OldPassword *string `json:"old_password"`
	NewPassword *string `json:"new_password"`
}

type UpdateUserInfoResponse struct {
	ID             uint64 `json:"id"`
	UserName       string `json:"user_name"`
	Email          string `json:"email"`
	UsedQuota      uint32 `json:"used_quota"`
	Quota          uint32 `json:"quota"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	RemainingQuota uint32 `json:"remaining_quota"`
	NeedRelogin    bool   `json:"need_relogin"`
}
