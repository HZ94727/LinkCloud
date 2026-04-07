package database

import (
	"context"
	"fmt"
	"time"

	"gitea.com/hz/linkcloud/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB    *gorm.DB
	Redis *redis.Client
	Ctx   = context.Background()
)

func Init() error {
	// 1. 初始化 MySQL
	if err := initMySQL(); err != nil {
		fmt.Println("init mysql failed")
		return err
	}

	// 2. 初始化 Redis
	if err := initRedis(); err != nil {
		fmt.Println("init redis failed")
		return err
	}

	return nil
}

func initMySQL() error {
	cfg := config.AppConfig.DB

	// DSN 连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	// 打开连接
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	// 获取底层 sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// 设置连接池参数（重要！）
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	DB = db
	return nil
}

func initRedis() error {
	cfg := config.AppConfig.Redis

	Redis = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password, // 没有密码则留空
		DB:           0,            // 使用默认 DB
		PoolSize:     50,           // 连接池大小
		MinIdleConns: 10,           // 最小空闲连接
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Redis.Ping(ctx).Err(); err != nil {
		return err
	}

	return nil
}

// 关闭连接（优雅退出时调用）
func Close() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		sqlDB.Close()
	}
	if Redis != nil {
		Redis.Close()
	}
}
