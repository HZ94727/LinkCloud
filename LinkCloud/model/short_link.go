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
