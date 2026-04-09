package repository

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ShortLinkRepository struct{}

func NewShortLinkRepository() *ShortLinkRepository {
	return &ShortLinkRepository{}
}

type ShortLinkListFilter struct {
	UserID     uint64
	Page       int
	PageSize   int
	SortBy     string
	SortOrder  string
	Status     *int8
	Keywords   map[string]string
	FuzzyQuery bool
}

type shortLinkCache struct {
	ID          uint64 `json:"id"`
	OriginalURL string `json:"original_url"`
	Password    string `json:"password"`
	Status      int8   `json:"status"`
	ExpireAt    *int64 `json:"expire_at"`
}

var allowedKeywordFields = map[string]struct{}{
	"short_code":   {},
	"original_url": {},
	"remark":       {},
	"domain":       {},
}

var allowedSortFields = map[string]struct{}{
	"created_at":  {},
	"click_count": {},
	"expire_at":   {},
	"updated_at":  {},
}

func (r *ShortLinkRepository) GetByShortCode(shortCode string) (*model.ShortLink, error) {
	var link model.ShortLink
	if err := database.DB.Where("short_code = ?", shortCode).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *ShortLinkRepository) GetOwnedByShortCode(userID uint64, shortCode string) (*model.ShortLink, error) {
	var link model.ShortLink
	if err := database.DB.Where("short_code = ? AND user_id = ?", shortCode, userID).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *ShortLinkRepository) ExistsByShortCode(shortCode string) (bool, error) {
	var count int64
	if err := database.DB.Model(&model.ShortLink{}).Where("short_code = ?", shortCode).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ShortLinkRepository) ListByUser(filter ShortLinkListFilter) ([]model.ShortLink, int64, error) {
	db := database.DB.Model(&model.ShortLink{}).
		Where("user_id = ?", filter.UserID)

	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}

	for field, value := range filter.Keywords {
		if value == "" {
			continue
		}
		if _, ok := allowedKeywordFields[field]; !ok {
			continue
		}
		if filter.FuzzyQuery {
			db = db.Where(field+" LIKE ?", "%"+value+"%")
		} else {
			db = db.Where(field+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := filter.SortBy
	if _, ok := allowedSortFields[sortBy]; !ok {
		sortBy = "created_at"
	}
	if filter.SortOrder == "asc" {
		db = db.Order(fmt.Sprintf("%s ASC", sortBy))
	} else {
		db = db.Order(fmt.Sprintf("%s DESC", sortBy))
	}

	offset := (filter.Page - 1) * filter.PageSize
	if offset < 0 {
		offset = 0
	}

	var links []model.ShortLink
	if err := db.Offset(offset).Limit(filter.PageSize).Find(&links).Error; err != nil {
		return nil, 0, err
	}

	return links, total, nil
}

func (r *ShortLinkRepository) Create(db *gorm.DB, link *model.ShortLink) error {
	if db == nil {
		db = database.DB
	}

	return db.Create(link).Error
}

func (r *ShortLinkRepository) Update(db *gorm.DB, link *model.ShortLink, updates map[string]any) error {
	if db == nil {
		db = database.DB
	}

	return db.Model(link).Updates(updates).Error
}

func (r *ShortLinkRepository) Delete(db *gorm.DB, link *model.ShortLink) error {
	if db == nil {
		db = database.DB
	}

	return db.Delete(link).Error
}

func (r *ShortLinkRepository) IncreaseClickCount(db *gorm.DB, id uint64) error {
	if db == nil {
		db = database.DB
	}

	return db.Model(&model.ShortLink{}).
		Where("id = ?", id).
		UpdateColumn("click_count", gorm.Expr("click_count + 1")).Error
}

func (r *ShortLinkRepository) GetCachedShortLink(shortCode string) (*model.ShortLink, bool, error) {
	cacheKey := fmt.Sprintf("link:%s", shortCode)
	cachedData, err := database.Redis.Get(database.Ctx, cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var cache shortLinkCache
	if err := json.Unmarshal(cachedData, &cache); err != nil {
		log.Printf("[WARN] Failed to unmarshal shortLink cache for %s: %v", shortCode, err)
		return nil, false, nil
	}

	link := &model.ShortLink{
		ID:          cache.ID,
		ShortCode:   shortCode,
		OriginalURL: cache.OriginalURL,
		Password:    cache.Password,
		Status:      cache.Status,
	}
	if cache.ExpireAt != nil {
		expireTime := time.Unix(*cache.ExpireAt, 0)
		link.ExpireAt = &expireTime
	}

	return link, true, nil
}

func (r *ShortLinkRepository) SetCachedShortLink(link model.ShortLink) {
	cacheKey := fmt.Sprintf("link:%s", link.ShortCode)

	cacheData := shortLinkCache{
		ID:          link.ID,
		OriginalURL: link.OriginalURL,
		Password:    link.Password,
		Status:      link.Status,
		ExpireAt:    nil,
	}

	if link.ExpireAt != nil {
		expireTimestamp := link.ExpireAt.Unix()
		cacheData.ExpireAt = &expireTimestamp
	}

	jsonData, err := json.Marshal(cacheData)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal cache data for shortCode %s: %v", link.ShortCode, err)
		return
	}

	var duration time.Duration
	if link.ExpireAt != nil {
		duration = time.Until(*link.ExpireAt)
		if duration <= 0 {
			log.Printf("[WARN] ExpireAt is in the past for shortCode %s, skip caching", link.ShortCode)
			return
		}
	} else {
		duration = 24 * time.Hour
	}

	if err := database.Redis.Set(database.Ctx, cacheKey, jsonData, duration).Err(); err != nil {
		log.Printf("[ERROR] Failed to cache shortLink %s to Redis: %v", link.ShortCode, err)
	} else {
		log.Printf("[INFO] Cached shortLink %s to Redis, expires in %v", link.ShortCode, duration)
	}
}

func (r *ShortLinkRepository) DeleteCachedShortLink(shortCode string) error {
	cacheKey := fmt.Sprintf("link:%s", shortCode)
	return database.Redis.Del(database.Ctx, cacheKey).Err()
}

func (r *ShortLinkRepository) EnsureAccessLogTable(tableName string) error {
	if !database.DB.Migrator().HasTable(tableName) {
		sql := `
    CREATE TABLE IF NOT EXISTS ` + tableName + ` (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        short_code_id BIGINT COMMENT '短码ID',
        access_time DATETIME NOT NULL COMMENT '访问时间',
        ip VARCHAR(45) COMMENT '访客IP',
        device_type VARCHAR(50) COMMENT '设备类型',
        referer VARCHAR(500) COMMENT '来源页面',
        user_agent VARCHAR(500) COMMENT 'User-Agent',
        status TINYINT DEFAULT 0 COMMENT '状态 0失败 1成功',
        error_message VARCHAR(255) DEFAULT '' COMMENT '错误信息',
        INDEX idx_short_code_id (short_code_id),
        INDEX idx_access_time (access_time)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='访问日志表'
    `
		return database.DB.Exec(sql).Error
	}
	return nil
}

func (r *ShortLinkRepository) SaveAccessLog(tableName string, data map[string]any) {
	go func() {
		if err := r.EnsureAccessLogTable(tableName); err != nil {
			log.Printf("[ERROR] 创建日志表失败: %v", err)
			return
		}

		if err := database.DB.Table(tableName).Create(data).Error; err != nil {
			log.Printf("[ERROR] 保存访问日志失败: %v", err)
		}
	}()
}
