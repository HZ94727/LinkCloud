package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/model"
	"gitea.com/hz/linkcloud/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// saveAccessLog 保存访问日志（异步）
func saveAccessLog(tableName string, data map[string]any) {
	go func() {
		// 确保表存在（可选，如果表已存在可以跳过）
		if !database.DB.Migrator().HasTable(tableName) {
			// 如果表不存在，创建表
			if err := createAccessLogTable(tableName); err != nil {
				log.Printf("[ERROR] 创建日志表失败: %v", err)
				return
			}
		}

		// 插入数据
		if err := database.DB.Table(tableName).Create(data).Error; err != nil {
			log.Printf("[ERROR] 保存访问日志失败: %v", err)
		}
	}()
}

// createAccessLogTable 创建日志表
func createAccessLogTable(tableName string) error {
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

func getDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "Mobile"
	}
	if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") {
		return "Tablet"
	}
	return "PC"
}

// Redirect 重定向短链接
func Redirect(c *gin.Context) {
	nowUTC := time.Now().UTC()
	year, month, _ := nowUTC.Date()
	// 创建表名
	tableName := fmt.Sprintf("access_logs_%d%02d", year, month)

	// 获取客户端信息
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	referer := c.Request.Referer()
	deviceType := getDeviceType(userAgent)

	// 获取短码
	shortCode := c.Param("short_code")
	// 获取密码
	password := c.Query("password")

	// 准备日志数据
	logData := map[string]any{
		"short_code_id": nil,
		"access_time":   nowUTC,
		"ip":            clientIP,
		"device_type":   deviceType,
		"referer":       referer,
		"user_agent":    userAgent,
		"status":        0, // 默认失败
		"error_message": nil,
	}

	cacheKey := fmt.Sprintf("link:%s", shortCode)
	var link model.ShortLink
	hitCache := false

	// 1. 先从 Redis 缓存获取
	cachedData, err := database.Redis.Get(database.Ctx, cacheKey).Bytes()
	if err == nil {
		// 缓存命中，反序列化
		var cache struct {
			ID          uint64 `json:"id"`
			OriginalURL string `json:"original_url"`
			Password    string `json:"password"`
			Status      int8   `json:"status"`
			ExpireAt    *int64 `json:"expire_at"`
		}
		if err := json.Unmarshal(cachedData, &cache); err == nil {
			link.ID = cache.ID
			link.OriginalURL = cache.OriginalURL
			link.Password = cache.Password
			link.Status = cache.Status
			link.ShortCode = shortCode
			if cache.ExpireAt != nil {
				expireTime := time.Unix(*cache.ExpireAt, 0)
				link.ExpireAt = &expireTime
			}
			hitCache = true
		}
	}

	// 2. 缓存未命中，从数据库查询
	if !hitCache {
		result := database.DB.Where("short_code = ?", shortCode).First(&link)
		if result.Error != nil {
			logData["error_message"] = "短链接不存在"
			saveAccessLog(tableName, logData)
			c.JSON(http.StatusOK, gin.H{
				"code":    -1,
				"message": "短链接不存在",
			})
			return
		}

		// 异步写入缓存
		go cacheShortLinkToRedis(link)
	}

	// 更新日志中的短链接ID
	logData["short_code_id"] = link.ID

	// 3. 检查过期时间
	if link.ExpireAt != nil && nowUTC.After(*link.ExpireAt) {
		logData["error_message"] = "短链接已过期"
		saveAccessLog(tableName, logData)
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "短链接已过期",
		})
		return
	}

	// 4. 检查禁用状态
	if link.Status == 0 {
		logData["error_message"] = "短链接已被禁用"
		saveAccessLog(tableName, logData)
		c.JSON(http.StatusOK, gin.H{
			"code":    -3,
			"message": "短链接已被禁用",
		})
		return
	}

	// 5. 密码验证
	if len(link.Password) != 0 {
		if len(password) == 0 {
			logData["error_message"] = "需要密码访问"
			saveAccessLog(tableName, logData)
			c.JSON(http.StatusOK, gin.H{
				"code":    -4,
				"message": "该链接需要密码访问",
			})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(link.Password), []byte(password)); err != nil {
			logData["error_message"] = "密码错误"
			saveAccessLog(tableName, logData)
			c.JSON(http.StatusOK, gin.H{
				"code":    -5,
				"message": "密码错误",
			})
			return
		}
	}

	// 6. 访问成功
	logData["status"] = 1
	logData["error_message"] = nil
	saveAccessLog(tableName, logData)

	// 7. 异步更新点击量
	go func() {
		if err := database.DB.Model(&model.ShortLink{}).
			Where("id = ?", link.ID).
			UpdateColumn("click_count", gorm.Expr("click_count + 1")).Error; err != nil {
			log.Printf("[ERROR] 更新点击量失败: %v", err)
		}
	}()

	// 8. 重定向
	c.Redirect(http.StatusFound, link.OriginalURL)
}

// cacheShortLinkToRedis 缓存短链接到 Redis
func cacheShortLinkToRedis(link model.ShortLink) {
	cacheKey := fmt.Sprintf("link:%s", link.ShortCode)

	cacheData := map[string]any{
		"id":           link.ID,
		"original_url": link.OriginalURL,
		"password":     link.Password,
		"status":       link.Status,
		"expire_at":    nil,
	}

	if link.ExpireAt != nil {
		expireTimestamp := link.ExpireAt.Unix()
		cacheData["expire_at"] = &expireTimestamp
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

	err = database.Redis.Set(database.Ctx, cacheKey, jsonData, duration).Err()
	if err != nil {
		log.Printf("[ERROR] Failed to cache shortLink %s to Redis: %v", link.ShortCode, err)
	} else {
		log.Printf("[INFO] Cached shortLink %s to Redis, expires in %v", link.ShortCode, duration)
	}
}

func CreateShortLink(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(200, gin.H{
			"code":    -1,
			"message": "未登录",
		})
		return
	}

	var req struct {
		OriginalURL string `json:"original_url" binding:"required"`
		Remark      string `json:"remark"`
		Password    string `json:"password"`
		ExpireAt    int64  `json:"expire_at"` // 过期时间戳（10位）
		Domain      string `json:"domain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    -2,
			"message": "请求参数不完整",
		})
		return
	}

	// 验证原始链接格式
	if !utils.IsValidURL(req.OriginalURL) {
		c.JSON(200, gin.H{
			"code":    -3,
			"message": "原始链接格式不正确",
		})
		return
	}

	// 检查用户配额
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(200, gin.H{
			"code":    -4,
			"message": "用户不存在",
		})
		return
	}

	if user.UsedQuota >= user.Quota {
		c.JSON(200, gin.H{
			"code":    -5,
			"message": "配额不足, 请充值",
		})
		return
	}

	// 生成唯一短码（最多重试3次）
	var shortCode string
	for i := range 3 {
		shortCode = generateShortCode()

		// 检查短码是否已存在
		var existLink model.ShortLink
		err := database.DB.Where("short_code = ?", shortCode).First(&existLink).Error
		if err != nil {
			// 没找到，说明短码可用
			break
		}
		// 重试三次也没与成功
		if i == 2 {
			c.JSON(200, gin.H{
				"code":    -6,
				"message": "短码生成失败, 请重试",
			})
			return
		}
	}

	// 处理过期时间
	var expireAt *time.Time
	if req.ExpireAt > 0 {
		t := time.Unix(req.ExpireAt, 0)
		expireAt = &t
	}

	// 处理密码（不为空则加密）
	var hashedPassword string
	if req.Password != "" {
		hashedPwd, err := utils.HashPassword(req.Password)
		if err != nil {
			c.JSON(200, gin.H{
				"code":    -7,
				"message": "系统繁忙, 请稍后再试",
			})
			return
		}
		hashedPassword = hashedPwd
	}

	// 创建短链接记录
	shortLink := model.ShortLink{
		ShortCode:   shortCode,
		OriginalURL: req.OriginalURL,
		UserID:      userID.(uint64),
		Remark:      req.Remark,
		Status:      1,
		Password:    hashedPassword,
		ExpireAt:    expireAt,
		ClickCount:  0,
		Domain:      req.Domain,
	}

	if err := database.DB.Create(&shortLink).Error; err != nil {
		c.JSON(200, gin.H{
			"code":    -7,
			"message": "生成短链接失败, 请稍后再试",
		})
		return
	}

	// 更新用户已使用配额
	database.DB.Model(&user).Update("used_quota", user.UsedQuota+1)

	// 返回成功响应
	c.JSON(200, gin.H{
		"code":    0,
		"message": "生成成功",
		"data": gin.H{
			"id":           shortLink.ID,
			"short_code":   shortLink.ShortCode,
			"short_url":    fmt.Sprintf("%s/%s", req.Domain, shortCode),
			"original_url": shortLink.OriginalURL,
			"remark":       shortLink.Remark,
			"status":       shortLink.Status,
			"has_password": req.Password != "",
			"expire_at":    req.ExpireAt,
			"click_count":  shortLink.ClickCount,
			"created_at":   shortLink.CreatedAt.Unix(),
			"updated_at":   shortLink.UpdatedAt.Unix(),
		},
	})

	go cacheShortLinkToRedis(shortLink)
}

// 生成6位随机短码（字母+数字）
func generateShortCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())

	code := make([]byte, 6)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

// GetShortLinks 获取短链接列表
func GetShortLinks(c *gin.Context) {

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 1. 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 2. 获取筛选条件
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// 3. 获取关键字搜索参数
	keywordMap := c.QueryMap("keyword") // keyword[remark]=博客&keyword[original_url]=deepseek
	fmt.Println("keywordmap is: ", keywordMap)
	fuzzyQuery := c.DefaultQuery("fuzzy_query", "true") // 是否模糊匹配，默认 true

	// 4. 构建查询
	query := database.DB.Model(&model.ShortLink{}).Where("user_id = ?", userID)

	// 5. 状态筛选
	if status, ok, err := parseBinaryStatus(c.Query("status")); err == nil && ok {
		query = query.Where("status = ?", status)
	}

	// 6. 关键字搜索（按字段）
	if len(keywordMap) > 0 {

		for field, value := range keywordMap {
			if value == "" {
				continue
			}

			// 根据 fuzzy_query 决定使用 LIKE 还是 =
			if fuzzyQuery == "true" {
				// 模糊匹配
				query = query.Where(field+" LIKE ?", "%"+value+"%")
			} else {
				// 精确匹配
				query = query.Where(field+" = ?", value)
			}
		}
	}

	// 7. 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "查询失败",
		})
		return
	}

	// 8. 排序
	validSortFields := map[string]bool{
		"created_at":  true,
		"click_count": true,
		"expire_at":   true,
		"updated_at":  true,
	}
	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder == "asc" {
		query = query.Order(fmt.Sprintf("%s ASC", sortBy))
	} else {
		query = query.Order(fmt.Sprintf("%s DESC", sortBy))
	}

	// 9. 分页查询
	var shortLinks []model.ShortLink
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&shortLinks).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "查询失败",
		})
		return
	}

	// 10. 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	// 11. 构建返回数据
	items := buildShortLinkItems(shortLinks, c.Request.Host)

	// 12. 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
			"items":       items,
		},
	})
}

// buildShortLinkItems 构建返回数据（复用代码）
func buildShortLinkItems(links []model.ShortLink, host string) []gin.H {
	items := make([]gin.H, 0, len(links))
	for _, link := range links {
		item := gin.H{
			"id":           link.ID,
			"short_code":   link.ShortCode,
			"short_url":    fmt.Sprintf("%s/%s", host, link.ShortCode),
			"original_url": link.OriginalURL,
			"remark":       link.Remark,
			"status":       link.Status,
			"has_password": link.Password != "",
			"click_count":  link.ClickCount,
			"created_at":   link.CreatedAt.Unix(),
			"updated_at":   link.UpdatedAt.Unix(),
		}

		if link.ExpireAt != nil {
			item["expire_at"] = link.ExpireAt.Unix()
			item["is_expired"] = time.Now().After(*link.ExpireAt)
		} else {
			item["expire_at"] = nil
			item["is_expired"] = false
		}

		items = append(items, item)
	}
	return items
}

// GetShortLink 获取短链接详情
func GetShortLink(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 2. 获取短码
	shortCode := c.Param("short_code")
	if shortCode == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "短码不能为空",
		})
		return
	}

	// 3. 查询短链接
	var shortLink model.ShortLink
	result := database.DB.Where("short_code = ? AND user_id = ? AND deleted_at IS NULL",
		shortCode, userID).First(&shortLink)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -3,
			"message": "短链接不存在",
		})
		return
	}

	// 4. 返回详情
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	shortURL := fmt.Sprintf("%s://%s/%s", scheme, c.Request.Host, shortLink.ShortCode)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": gin.H{
			"id":           shortLink.ID,
			"short_code":   shortLink.ShortCode,
			"short_url":    shortURL,
			"original_url": shortLink.OriginalURL,
			"remark":       shortLink.Remark,
			"status":       shortLink.Status,
			"has_password": shortLink.Password != "",
			"click_count":  shortLink.ClickCount,
			"expire_at":    getExpireAtTimestamp(shortLink.ExpireAt),
			"is_expired":   isExpired(shortLink.ExpireAt),
			"created_at":   shortLink.CreatedAt.Unix(),
			"updated_at":   shortLink.UpdatedAt.Unix(),
		},
	})
}

// getExpireAtTimestamp 获取过期时间戳（处理 nil）
func getExpireAtTimestamp(expireAt *time.Time) interface{} {
	if expireAt == nil {
		return nil
	}
	return expireAt.Unix()
}

// isExpired 判断是否已过期
func isExpired(expireAt *time.Time) bool {
	if expireAt == nil {
		return false
	}
	return time.Now().After(*expireAt)
}

// UpdateShortLink 更新短链接
func UpdateShortLink(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 2. 获取短码
	shortCode := c.Param("short_code")

	// 短码为空字符串 "" 的情况
	if shortCode == "\"\"" {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "短码不能为空",
		})
		return
	}

	// 3. 绑定请求体
	var req struct {
		OriginalURL *string `json:"original_url"`
		Remark      *string `json:"remark"`
		Password    *string `json:"password"`
		ExpireAt    *int64  `json:"expire_at"`
		Status      *int8   `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -3,
			"message": "参数错误",
		})
		return
	}

	// 4. 查询短链接（验证所有权）
	var shortLink model.ShortLink
	result := database.DB.Where("short_code = ? AND user_id = ? AND deleted_at IS NULL", shortCode, userID).First(&shortLink)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -4,
			"message": "短链接不存在",
		})
		return
	}

	// 5. 构建更新数据
	updates := make(map[string]interface{})

	// 原始链接
	if req.OriginalURL != nil {
		if *req.OriginalURL == "" {
			c.JSON(http.StatusOK, gin.H{
				"code":    -5,
				"message": "原始链接不能为空",
			})
			return
		}
		updates["original_url"] = *req.OriginalURL
	}

	// 备注
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	// 密码（需要加密）
	if req.Password != nil {
		if *req.Password == "" {
			// 空字符串表示清除密码
			updates["password"] = ""
		} else {
			// 加密新密码
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"code":    -6,
					"message": "系统繁忙, 请稍后再试",
				})
				return
			}
			updates["password"] = string(hashedPassword)
		}
	}

	// 过期时间
	if req.ExpireAt != nil {
		if *req.ExpireAt == 0 {
			// 传 0 表示清除过期时间
			updates["expire_at"] = nil
		} else {
			expireTime := time.Unix(*req.ExpireAt, 0)
			// 校验过期时间不能早于当前时间
			if expireTime.Before(time.Now()) {
				c.JSON(http.StatusOK, gin.H{
					"code":    -7,
					"message": "过期时间不能早于当前时间",
				})
				return
			}
			updates["expire_at"] = expireTime
		}
	}

	// 状态
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			c.JSON(http.StatusOK, gin.H{
				"code":    -8,
				"message": "状态值无效, 只能为0或1",
			})
			return
		}
		updates["status"] = *req.Status
	}

	// 6. 执行更新
	if len(updates) > 0 {
		if err := database.DB.Model(&shortLink).Updates(updates).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    -9,
				"message": "更新失败",
			})
			return
		}
	}

	// 7. 删除 Redis 缓存
	cacheKey := fmt.Sprintf("link:%s", shortCode)
	database.Redis.Del(database.Ctx, cacheKey)

	// 8. 重新查询更新后的数据
	var updatedLink model.ShortLink
	database.DB.Where("short_code = ?", shortCode).First(&updatedLink)

	// 9. 异步重新写入缓存
	go cacheShortLinkToRedis(updatedLink)

	// 10. 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"id":           updatedLink.ID,
			"short_code":   updatedLink.ShortCode,
			"original_url": updatedLink.OriginalURL,
			"remark":       updatedLink.Remark,
			"status":       updatedLink.Status,
			"has_password": updatedLink.Password != "",
			"expire_at":    getExpireAtTimestamp(updatedLink.ExpireAt),
			"updated_at":   updatedLink.UpdatedAt.Unix(),
		},
	})
}

// DeleteShortLink 删除短链接（软删除）
func DeleteShortLink(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 2. 获取短码
	shortCode := c.Param("short_code")
	if shortCode == `""` {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "短码不能为空",
		})
		return
	}

	// 3. 查询短链接（验证所有权）
	var shortLink model.ShortLink
	result := database.DB.Where("short_code = ? AND user_id = ? AND deleted_at IS NULL",
		shortCode, userID).First(&shortLink)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -3,
			"message": "短链接不存在",
		})
		return
	}

	// 4. 软删除（GORM 自动设置 deleted_at）
	if err := database.DB.Delete(&shortLink).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -4,
			"message": "删除失败",
		})
		return
	}

	// 5. 删除 Redis 缓存
	cacheKey := fmt.Sprintf("link:%s", shortCode)
	database.Redis.Del(database.Ctx, cacheKey)

	// 6. 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}
