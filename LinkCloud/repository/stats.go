package repository

import (
	"fmt"
	"strings"
	"time"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/model"
)

type StatsRepository struct{}

func NewStatsRepository() *StatsRepository {
	return &StatsRepository{}
}

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
	Status     int8      `gorm:"column:status"`
	ErrorMsg   string    `gorm:"column:error_message"`
}

func (r *StatsRepository) GetShortLink(shortCode string, userID uint64) (*model.ShortLink, error) {
	return NewShortLinkRepository().GetOwnedByShortCode(userID, shortCode)
}

func (r *StatsRepository) QueryStats(shortLinkID uint64, startAt, endAt time.Time, tzOffset string) (*dto.StatsResponse, error) {
	tables := existingAccessLogTables(startAt, endAt)
	if len(tables) == 0 {
		return &dto.StatsResponse{
			TotalClicks:    0,
			UniqueVisitors: 0,
			DailyClicks:    []dto.StatsCountItem{},
			DeviceStats:    []dto.DeviceStatItem{},
			Referers:       []dto.RefererStatItem{},
		}, nil
	}

	unionSQL, unionArgs := buildAccessLogUnionSQL(tables, shortLinkID, startAt, endAt)

	var totalRow statsCountRow
	if err := database.DB.Raw("SELECT COUNT(1) AS count FROM ("+unionSQL+") AS logs", unionArgs...).Scan(&totalRow).Error; err != nil {
		return nil, err
	}

	var uvRow statsCountRow
	if err := database.DB.Raw("SELECT COUNT(DISTINCT ip) AS count FROM ("+unionSQL+") AS logs", unionArgs...).Scan(&uvRow).Error; err != nil {
		return nil, err
	}

	var dailyRows []dailyClickRow
	dailySQL := "SELECT date, COUNT(1) AS count FROM (" +
		"SELECT DATE_FORMAT(CONVERT_TZ(access_time, '+00:00', ?), '%Y-%m-%d') AS date FROM (" + unionSQL + ") AS logs" +
		") AS daily_logs GROUP BY date ORDER BY date ASC"
	dailyArgs := make([]any, 0, len(unionArgs)+1)
	dailyArgs = append(dailyArgs, tzOffset)
	dailyArgs = append(dailyArgs, unionArgs...)
	if err := database.DB.Raw(dailySQL, dailyArgs...).Scan(&dailyRows).Error; err != nil {
		return nil, err
	}

	var deviceRows []deviceStatsRow
	if err := database.DB.Raw(
		"SELECT COALESCE(NULLIF(device_type, ''), 'Unknown') AS device_type, COUNT(1) AS count FROM ("+unionSQL+") AS logs GROUP BY COALESCE(NULLIF(device_type, ''), 'Unknown') ORDER BY count DESC",
		unionArgs...,
	).Scan(&deviceRows).Error; err != nil {
		return nil, err
	}

	var refererRows []refererStatsRow
	if err := database.DB.Raw(
		"SELECT COALESCE(NULLIF(referer, ''), '直接访问') AS referer, COUNT(1) AS count FROM ("+unionSQL+") AS logs GROUP BY COALESCE(NULLIF(referer, ''), '直接访问') ORDER BY count DESC LIMIT 10",
		unionArgs...,
	).Scan(&refererRows).Error; err != nil {
		return nil, err
	}

	return &dto.StatsResponse{
		TotalClicks:    totalRow.Count,
		UniqueVisitors: uvRow.Count,
		DailyClicks:    buildDailyClickItems(dailyRows),
		DeviceStats:    buildDeviceStatItems(deviceRows),
		Referers:       buildRefererStatItems(refererRows),
	}, nil
}

func (r *StatsRepository) QueryLogs(shortLinkID uint64, startAt, endAt time.Time, page, size int) (*dto.LogListResponse, error) {
	tables := existingAccessLogTables(startAt, endAt)
	if len(tables) == 0 {
		return &dto.LogListResponse{
			Total:      0,
			Page:       page,
			PageSize:   size,
			TotalPages: 0,
			List:       []dto.AccessLogItem{},
		}, nil
	}

	unionSQL, unionArgs := buildAccessLogUnionSQL(tables, shortLinkID, startAt, endAt)

	var totalRow statsCountRow
	if err := database.DB.Raw("SELECT COUNT(1) AS count FROM ("+unionSQL+") AS logs", unionArgs...).Scan(&totalRow).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * size
	queryArgs := append([]any{}, unionArgs...)
	queryArgs = append(queryArgs, size, offset)

	var rows []accessLogRow
	if err := database.DB.Raw(
		"SELECT access_time, ip, device_type, referer, user_agent, status, error_message FROM ("+unionSQL+") AS logs ORDER BY access_time DESC LIMIT ? OFFSET ?",
		queryArgs...,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	totalPages := int(totalRow.Count) / size
	if int(totalRow.Count)%size != 0 {
		totalPages++
	}

	return &dto.LogListResponse{
		Total:      totalRow.Count,
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
		List:       buildAccessLogItems(rows),
	}, nil
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

func buildDailyClickItems(rows []dailyClickRow) []dto.StatsCountItem {
	items := make([]dto.StatsCountItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.StatsCountItem{
			Date:  row.Date,
			Count: row.Count,
		})
	}
	return items
}

func buildDeviceStatItems(rows []deviceStatsRow) []dto.DeviceStatItem {
	items := make([]dto.DeviceStatItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.DeviceStatItem{
			DeviceType: row.DeviceType,
			Count:      row.Count,
		})
	}
	return items
}

func buildRefererStatItems(rows []refererStatsRow) []dto.RefererStatItem {
	items := make([]dto.RefererStatItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.RefererStatItem{
			Refer: row.Referer,
			Count: row.Count,
		})
	}
	return items
}

func buildAccessLogItems(rows []accessLogRow) []dto.AccessLogItem {
	items := make([]dto.AccessLogItem, 0, len(rows))
	localLocation := time.Now().Location()
	for _, row := range rows {
		items = append(items, dto.AccessLogItem{
			AccessTime:   row.AccessTime.In(localLocation).Format("2006-01-02 15:04:05"),
			IP:           row.IP,
			DeviceType:   row.DeviceType,
			Referer:      row.Referer,
			Browser:      parseBrowserType(row.UserAgent),
			Status:       row.Status,
			ErrorMessage: row.ErrorMsg,
		})
	}
	return items
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
