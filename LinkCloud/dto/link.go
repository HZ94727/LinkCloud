package dto

type CreateShortLinkRequest struct {
	OriginalURL string `json:"original_url" binding:"required"`
	Remark      string `json:"remark"`
	Password    string `json:"password"`
	ExpireAt    int64  `json:"expire_at"`
	Domain      string `json:"domain" binding:"required"`
}

type CreateShortLinkResponse struct {
	ID          uint64 `json:"id"`
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	Remark      string `json:"remark"`
	Status      int8   `json:"status"`
	HasPassword bool   `json:"has_password"`
	ExpireAt    int64  `json:"expire_at"`
	ClickCount  uint32 `json:"click_count"`
	CreatedAt   int64  `json:"created_at"`
}

type ShortLinkListRequest struct {
	Page       int
	PageSize   int
	SortBy     string
	SortOrder  string
	Status     *int8
	Keywords   map[string]string
	FuzzyQuery bool
}

type ShortLinkListQuery struct {
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
	SortBy     string `form:"sort_by,default=created_at"`
	SortOrder  string `form:"sort_order,default=desc"`
	FuzzyQuery bool   `form:"fuzzy_query,default=true"`
	Status     *int8  `form:"status" binding:"omitempty,oneof=0 1"`
}

type ShortLinkListItem struct {
	ID          uint64 `json:"id"`
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	Remark      string `json:"remark"`
	Status      int8   `json:"status"`
	HasPassword bool   `json:"has_password"`
	ClickCount  uint32 `json:"click_count"`
	ExpireAt    *int64 `json:"expire_at"`
	IsExpired   bool   `json:"is_expired"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type ShortLinkListResponse struct {
	Items      []ShortLinkListItem `json:"items"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

type ShortLinkDetailResponse struct {
	ID          uint64 `json:"id"`
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	Remark      string `json:"remark"`
	Status      int8   `json:"status"`
	HasPassword bool   `json:"has_password"`
	ClickCount  uint32 `json:"click_count"`
	ExpireAt    *int64 `json:"expire_at"`
	IsExpired   bool   `json:"is_expired"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type UpdateShortLinkRequest struct {
	OriginalURL *string `json:"original_url"`
	Remark      *string `json:"remark"`
	Password    *string `json:"password"`
	ExpireAt    *int64  `json:"expire_at"`
	Status      *int8   `json:"status"`
}
