package model

import (
	"time"

	"gorm.io/gorm"
)

type ShortLink struct {
	ID          uint64     `gorm:"primaryKey"`
	ShortCode   string     `gorm:"unique;size:20;not null"`
	OriginalURL string     `gorm:"type:text;not null"`
	UserID      uint64     `gorm:"not null;index"`
	Remark      string     `gorm:"size:100"`
	Status      int8       `gorm:"default:1"` // 1启用 0禁用
	Password    string     `gorm:"size:255"`  // 空表示无密码
	ExpireAt    *time.Time // NULL表示永不过期
	ClickCount  uint32     `gorm:"default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Domain      string
	DeletedAt   gorm.DeletedAt
}

func (ShortLink) TableName() string {
	return "short_links"
}

// 创建钩子函数，在写入数据库的时候，将时间精确到秒
func (link *ShortLink) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Truncate(time.Second)
	link.CreatedAt = now
	// link.UpdatedAt = now
	return nil
}

func (link *ShortLink) BeforeUpdate(tx *gorm.DB) error {
	// 注意：Updates(map) 不会使用这个值，需要在 Repository 层手动设置
	// link.UpdatedAt = time.Now().Truncate(time.Second)
	return nil
}
