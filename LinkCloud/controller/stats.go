package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/model"
	"github.com/gin-gonic/gin"
)

type statsCountRow struct {
	Count int64 `gorm:"column:count"`
}

type dailyClickRow struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}

type deviceStatsRow struct {
	DeviceType string `gorm:"column:device_type"`
	Count      int64  `gorm:"column:count"`
}

type refererStatsRow struct {
	Referer string `gorm:"column:referer"`
	Count   int64  `gorm:"column:count"`
}

type accessLogRow struct {
	AccessTime time.Time `gorm:"column:access_time"`
	IP         string    `gorm:"column:ip"`
	DeviceType string    `gorm:"column:device_type"`
	Referer    string    `gorm:"column:referer"`
	UserAgent  string    `gorm:"column:user_agent"`
	Status     int       `gorm:"column:status"`
	ErrorMsg   string    `gorm:"column:error_message"`
}

func GetStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	shortLink, ok := getOwnedShortLink(c, userID)
	if !ok {
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	nowUTC := time.Now().UTC()
	startAt := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -days+1)
	endAt := nowUTC
	localTZOffset := formatTZOffset(time.Now())

	tables := existingAccessLogTables(startAt, endAt)
	if len(tables) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
			"data": gin.H{
				"total_clicks":    0,
				"unique_visitors": 0,
				"daily_clicks":    []gin.H{},
				"device_stats":    []gin.H{},
				"referers":        []gin.H{},
			},
		})
		return
	}

	unionSQL, unionArgs := buildAccessLogUnionSQL(tables, shortLink.ID, startAt, endAt)

	var totalRow statsCountRow
	if err := database.DB.Raw("SELECT COUNT(1) AS count FROM ("+unionSQL+") AS logs", unionArgs...).Scan(&totalRow).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -2, "message": "统计查询失败"})
		return
	}

	var uvRow statsCountRow
	if err := database.DB.Raw("SELECT COUNT(DISTINCT ip) AS count FROM ("+unionSQL+") AS logs", unionArgs...).Scan(&uvRow).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -2, "message": "统计查询失败"})
		return
	}

	var dailyRows []dailyClickRow
	dailySQL := "SELECT date, COUNT(1) AS count FROM (" +
		"SELECT DATE_FORMAT(CONVERT_TZ(access_time, '+00:00', ?), '%Y-%m-%d') AS date FROM (" + unionSQL + ") AS logs" +
		") AS daily_logs GROUP BY date ORDER BY date ASC"
	dailyArgs := make([]any, 0, len(unionArgs)+1)
	dailyArgs = append(dailyArgs, localTZOffset)
	dailyArgs = append(dailyArgs, unionArgs...)
	if err := database.DB.Raw(
		dailySQL,
		dailyArgs...,
	).Scan(&dailyRows).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -2, "message": "统计查询失败"})
		return
	}

	var deviceRows []deviceStatsRow
	if err := database.DB.Raw(
		"SELECT COALESCE(NULLIF(device_type, ''), 'Unknown') AS device_type, COUNT(1) AS count FROM ("+unionSQL+") AS logs GROUP BY COALESCE(NULLIF(device_type, ''), 'Unknown') ORDER BY count DESC",
		unionArgs...,
	).Scan(&deviceRows).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -2, "message": "统计查询失败"})
		return
	}

	var refererRows []refererStatsRow
	if err := database.DB.Raw(
		"SELECT COALESCE(NULLIF(referer, ''), '直接访问') AS referer, COUNT(1) AS count FROM ("+unionSQL+") AS logs GROUP BY COALESCE(NULLIF(referer, ''), '直接访问') ORDER BY count DESC LIMIT 10",
		unionArgs...,
	).Scan(&refererRows).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -2, "message": "统计查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"total_clicks":    totalRow.Count,
			"unique_visitors": uvRow.Count,
			"daily_clicks":    buildDailyClickItems(dailyRows),
			"device_stats":    buildDeviceStatItems(deviceRows),
			"referers":        buildRefererStatItems(refererRows),
		},
	})
}

func GetLogs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	shortLink, ok := getOwnedShortLink(c, userID)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	startAt, hasStart := parseUnixQuery(c.Query("start_at"))
	endAt, hasEnd := parseUnixQuery(c.Query("end_at"))
	if !hasStart {
		startAt = shortLink.CreatedAt.UTC()
	}
	if !hasEnd {
		endAt = time.Now().UTC()
	}
	if endAt.Before(startAt) {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "时间范围无效",
		})
		return
	}

	tables := existingAccessLogTables(startAt, endAt)
	if len(tables) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
			"data": gin.H{
				"total":       0,
				"page":        page,
				"page_size":   size,
				"list":        []gin.H{},
				"total_pages": 0,
			},
		})
		return
	}

	unionSQL, unionArgs := buildAccessLogUnionSQL(tables, shortLink.ID, startAt, endAt)

	var totalRow statsCountRow
	if err := database.DB.Raw("SELECT COUNT(1) AS count FROM ("+unionSQL+") AS logs", unionArgs...).Scan(&totalRow).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -3, "message": "查询访问日志失败"})
		return
	}

	offset := (page - 1) * size
	queryArgs := append([]any{}, unionArgs...)
	queryArgs = append(queryArgs, size, offset)

	var logs []accessLogRow
	if err := database.DB.Raw(
		"SELECT access_time, ip, device_type, referer, user_agent, status, error_message FROM ("+unionSQL+") AS logs ORDER BY access_time DESC LIMIT ? OFFSET ?",
		queryArgs...,
	).Scan(&logs).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -3, "message": "查询访问日志失败"})
		return
	}

	totalPages := int(totalRow.Count) / size
	if int(totalRow.Count)%size != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"total":       totalRow.Count,
			"page":        page,
			"page_size":   size,
			"list":        buildAccessLogItems(logs),
			"total_pages": totalPages,
		},
	})
}

func getOwnedShortLink(c *gin.Context, userID any) (model.ShortLink, bool) {
	shortCode := c.Param("short_code")
	if shortCode == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "短码不能为空",
		})
		return model.ShortLink{}, false
	}

	var shortLink model.ShortLink
	if err := database.DB.Where("short_code = ? AND user_id = ? AND deleted_at IS NULL", shortCode, userID).First(&shortLink).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "短链接不存在",
		})
		return model.ShortLink{}, false
	}

	return shortLink, true
}

func parseUnixQuery(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	ts, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ts <= 0 {
		return time.Time{}, false
	}
	return time.Unix(ts, 0).UTC(), true
}

func existingAccessLogTables(startAt, endAt time.Time) []string {
	tables := make([]string, 0)
	for current := monthStart(startAt); !current.After(monthStart(endAt)); current = current.AddDate(0, 1, 0) {
		tableName := fmt.Sprintf("access_logs_%04d%02d", current.Year(), int(current.Month()))
		if database.DB.Migrator().HasTable(tableName) {
			tables = append(tables, tableName)
		}
	}
	return tables
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func buildAccessLogUnionSQL(tables []string, shortCodeID uint64, startAt, endAt time.Time) (string, []any) {
	parts := make([]string, 0, len(tables))
	args := make([]any, 0, len(tables)*3)

	for _, table := range tables {
		parts = append(parts, fmt.Sprintf(
			"SELECT access_time, ip, device_type, referer, user_agent, status, error_message FROM `%s` WHERE short_code_id = ? AND access_time >= ? AND access_time <= ?",
			table,
		))
		args = append(args, shortCodeID, startAt, endAt)
	}

	return strings.Join(parts, " UNION ALL "), args
}

func buildDailyClickItems(rows []dailyClickRow) []gin.H {
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"date":  row.Date,
			"count": row.Count,
		})
	}
	return items
}

func buildDeviceStatItems(rows []deviceStatsRow) []gin.H {
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"device_type": row.DeviceType,
			"count":       row.Count,
		})
	}
	return items
}

func buildRefererStatItems(rows []refererStatsRow) []gin.H {
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"referer": row.Referer,
			"count":   row.Count,
		})
	}
	return items
}

func buildAccessLogItems(rows []accessLogRow) []gin.H {
	items := make([]gin.H, 0, len(rows))
	localLocation := time.Now().Location()
	for _, row := range rows {
		items = append(items, gin.H{
			"access_time":   row.AccessTime.In(localLocation).Format("2006-01-02 15:04:05"),
			"ip":            row.IP,
			"device_type":   row.DeviceType,
			"referer":       row.Referer,
			"browser":       parseBrowserType(row.UserAgent),
			"status":        row.Status,
			"error_message": row.ErrorMsg,
		})
	}
	return items
}

func formatTZOffset(t time.Time) string {
	_, offsetSeconds := t.Zone()
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

func parseBrowserType(userAgent string) string {
	ua := strings.ToLower(userAgent)

	switch {
	case strings.Contains(ua, "edg/"):
		return "Edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		return "Opera"
	case strings.Contains(ua, "chrome/") && !strings.Contains(ua, "edg/"):
		return "Chrome"
	case strings.Contains(ua, "firefox/"):
		return "Firefox"
	case strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome/") && !strings.Contains(ua, "chromium/"):
		return "Safari"
	case strings.Contains(ua, "msie") || strings.Contains(ua, "trident/"):
		return "IE"
	case ua == "":
		return "Unknown"
	default:
		return "Other"
	}
}
