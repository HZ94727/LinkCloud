package model

import "time"

type User struct {
	ID        uint64 `gorm:"primaryKey"`
	Email     string `gorm:"unique;size:100"`
	UserName  string `gorm:"unique;size:50"`
	Password  string `gorm:"size:255"`
	Status    int8   `gorm:"default:1"`
	Quota     uint32 `gorm:"default:100"`
	UsedQuota uint32 `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (User) TableName() string {
	return "users"
}
