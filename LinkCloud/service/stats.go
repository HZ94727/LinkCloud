package service

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/repository"
	"gorm.io/gorm"
)

type StatsService struct {
	statsRepo *repository.StatsRepository
}

func NewStatsService(statsRepo *repository.StatsRepository) *StatsService {
	return &StatsService{statsRepo: statsRepo}
}

func DefaultStatsService() *StatsService {
	return NewStatsService(repository.NewStatsRepository())
}

func (s *StatsService) GetStats(userID uint64, shortCode string, days int, baseTZ time.Time) (*dto.StatsResponse, int, string) {
	shortLink, err := s.statsRepo.GetShortLink(shortCode, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ecode.CodeShortLinkNotFound, ecode.Message(ecode.CodeShortLinkNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	days = normalizeDays(days)
	nowUTC := time.Now().UTC()
	startAt := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -days+1)

	resp, err := s.statsRepo.QueryStats(shortLink.ID, startAt, nowUTC, formatTZOffset(baseTZ))
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return resp, ecode.CodeOK, "ok"
}

func (s *StatsService) GetLogs(userID uint64, shortCode string, page, size int, startAtRaw, endAtRaw string) (*dto.LogListResponse, int, string) {
	shortLink, err := s.statsRepo.GetShortLink(shortCode, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ecode.CodeShortLinkNotFound, ecode.Message(ecode.CodeShortLinkNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	startAt, hasStart := parseUnixQuery(startAtRaw)
	endAt, hasEnd := parseUnixQuery(endAtRaw)
	if !hasStart {
		startAt = shortLink.CreatedAt.UTC()
	}
	if !hasEnd {
		endAt = time.Now().UTC()
	}
	if endAt.Before(startAt) {
		return nil, ecode.CodeTimeRangeInvalid, ecode.Message(ecode.CodeTimeRangeInvalid)
	}

	resp, err := s.statsRepo.QueryLogs(shortLink.ID, startAt, endAt, page, size)
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return resp, ecode.CodeOK, "ok"
}

// unix 时间戳转换为UTC时间
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

func normalizeDays(days int) int {
	if days <= 0 {
		return 7
	}
	if days > 90 {
		return 90
	}
	return days
}
