package service

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/model"
	"gitea.com/hz/linkcloud/repository"
	"gitea.com/hz/linkcloud/utils"
	"gorm.io/gorm"
)

type LinkService struct {
	userRepo *repository.UserRepository
	linkRepo *repository.ShortLinkRepository
}

func NewLinkService(userRepo *repository.UserRepository, linkRepo *repository.ShortLinkRepository) *LinkService {
	return &LinkService{
		userRepo: userRepo,
		linkRepo: linkRepo,
	}
}

func DefaultLinkService() *LinkService {
	return NewLinkService(repository.NewUserRepository(), repository.NewShortLinkRepository())
}

func (s *LinkService) CreateShortLink(userID uint64, req dto.CreateShortLinkRequest) (*dto.CreateShortLinkResponse, int, string) {
	if !utils.IsValidURL(req.OriginalURL) {
		return nil, ecode.CodeOriginalURLInvalid, ecode.Message(ecode.CodeOriginalURLInvalid)
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ecode.CodeUserNotFound, ecode.Message(ecode.CodeUserNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	if user.UsedQuota >= user.Quota {
		return nil, ecode.CodeQuotaInsufficient, ecode.Message(ecode.CodeQuotaInsufficient)
	}

	shortCode, err := s.generateUniqueShortCode()
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	var expireAt *time.Time
	if req.ExpireAt > 0 {
		t := time.Unix(req.ExpireAt, 0)
		expireAt = &t
	}

	hashedPassword := ""
	if req.Password != "" {
		hashedPwd, err := utils.HashPassword(req.Password)
		if err != nil {
			return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
		}
		hashedPassword = hashedPwd
	}

	shortLink := model.ShortLink{
		ShortCode:   shortCode,
		OriginalURL: req.OriginalURL,
		UserID:      userID,
		Remark:      req.Remark,
		Status:      1,
		Password:    hashedPassword,
		ExpireAt:    expireAt,
		ClickCount:  0,
		Domain:      req.Domain,
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.linkRepo.Create(tx, &shortLink); err != nil {
			return err
		}

		return s.userRepo.IncreaseUsedQuota(tx, userID, 1)
	}); err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	response := &dto.CreateShortLinkResponse{
		ID:          shortLink.ID,
		ShortCode:   shortLink.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", normalizeBaseURL(req.Domain), shortLink.ShortCode),
		OriginalURL: shortLink.OriginalURL,
		Remark:      shortLink.Remark,
		Status:      shortLink.Status,
		HasPassword: req.Password != "",
		ExpireAt:    req.ExpireAt,
		ClickCount:  shortLink.ClickCount,
		CreatedAt:   shortLink.CreatedAt.Unix(),
	}

	go s.linkRepo.SetCachedShortLink(shortLink)

	return response, ecode.CodeOK, "生成成功"
}

func (s *LinkService) ResolveShortLink(shortCode, password, clientIP, userAgent, referer string) (*model.ShortLink, int, string) {
	shortCode = strings.TrimSpace(shortCode)
	if shortCode == "" {
		return nil, ecode.CodeShortCodeEmpty, ecode.Message(ecode.CodeShortCodeEmpty)
	}

	nowUTC := time.Now().UTC()
	year, month, _ := nowUTC.Date()
	tableName := fmt.Sprintf("access_logs_%d%02d", year, month)
	deviceType := getDeviceType(userAgent)
	logData := map[string]any{
		"short_code_id": nil,
		"access_time":   nowUTC,
		"ip":            clientIP,
		"device_type":   deviceType,
		"referer":       referer,
		"user_agent":    userAgent,
		"status":        int8(0),
		"error_message": nil,
	}

	link, err := s.resolveShortLink(shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logData["error_message"] = ecode.Message(ecode.CodeShortLinkNotFound)
			s.linkRepo.SaveAccessLog(tableName, logData)
			return nil, ecode.CodeShortLinkNotFound, ecode.Message(ecode.CodeShortLinkNotFound)
		}

		logData["error_message"] = ecode.Message(ecode.CodeSystemBusy)
		s.linkRepo.SaveAccessLog(tableName, logData)
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	logData["short_code_id"] = link.ID

	if link.ExpireAt != nil && nowUTC.After(*link.ExpireAt) {
		logData["error_message"] = ecode.Message(ecode.CodeShortLinkExpired)
		s.linkRepo.SaveAccessLog(tableName, logData)
		return nil, ecode.CodeShortLinkExpired, ecode.Message(ecode.CodeShortLinkExpired)
	}

	if link.Status == 0 {
		logData["error_message"] = ecode.Message(ecode.CodeShortLinkDisabled)
		s.linkRepo.SaveAccessLog(tableName, logData)
		return nil, ecode.CodeShortLinkDisabled, ecode.Message(ecode.CodeShortLinkDisabled)
	}

	if link.Password != "" {
		if password == "" {
			logData["error_message"] = ecode.Message(ecode.CodeShortLinkNeedPassword)
			s.linkRepo.SaveAccessLog(tableName, logData)
			return nil, ecode.CodeShortLinkNeedPassword, ecode.Message(ecode.CodeShortLinkNeedPassword)
		}

		if !utils.CheckPasswordHash(password, link.Password) {
			logData["error_message"] = ecode.Message(ecode.CodeShortLinkPasswordBad)
			s.linkRepo.SaveAccessLog(tableName, logData)
			return nil, ecode.CodeShortLinkPasswordBad, ecode.Message(ecode.CodeShortLinkPasswordBad)
		}
	}

	logData["status"] = int8(1)
	logData["error_message"] = nil
	s.linkRepo.SaveAccessLog(tableName, logData)

	go func(linkID uint64) {
		if err := s.linkRepo.IncreaseClickCount(nil, linkID); err != nil {
			// 这里只记录日志，不影响跳转成功
			fmt.Printf("[ERROR] 更新点击量失败: %v\n", err)
		}
	}(link.ID)

	return link, ecode.CodeOK, "ok"
}

func (s *LinkService) ListShortLinks(userID uint64, req dto.ShortLinkListRequest, baseURL string) (*dto.ShortLinkListResponse, int, string) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	links, total, err := s.linkRepo.ListByUser(repository.ShortLinkListFilter{
		UserID:     userID,
		Page:       req.Page,
		PageSize:   req.PageSize,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Status:     req.Status,
		Keywords:   req.Keywords,
		FuzzyQuery: req.FuzzyQuery,
	})
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize != 0 {
		totalPages++
	}

	items := make([]dto.ShortLinkListItem, 0, len(links))
	for _, link := range links {
		items = append(items, buildShortLinkListItem(link, baseURL))
	}

	return &dto.ShortLinkListResponse{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, ecode.CodeOK, "获取成功"
}

func (s *LinkService) GetShortLinkDetail(userID uint64, shortCode, baseURL string) (*dto.ShortLinkDetailResponse, int, string) {
	shortCode = strings.TrimSpace(shortCode)
	if shortCode == "" {
		return nil, ecode.CodeShortCodeEmpty, ecode.Message(ecode.CodeShortCodeEmpty)
	}

	link, err := s.linkRepo.GetOwnedByShortCode(userID, shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ecode.CodeShortLinkNotFound, ecode.Message(ecode.CodeShortLinkNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return buildShortLinkDetailResponse(*link, baseURL), ecode.CodeOK, "获取成功"
}

func (s *LinkService) UpdateShortLink(userID uint64, shortCode string, req dto.UpdateShortLinkRequest, baseURL string) (*dto.ShortLinkDetailResponse, int, string) {
	shortCode = strings.TrimSpace(shortCode)
	if shortCode == "" {
		return nil, ecode.CodeShortCodeEmpty, ecode.Message(ecode.CodeShortCodeEmpty)
	}

	shortLink, err := s.linkRepo.GetOwnedByShortCode(userID, shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ecode.CodeShortLinkNotFound, ecode.Message(ecode.CodeShortLinkNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	updates := make(map[string]any)

	if req.OriginalURL != nil {
		if !utils.IsValidURL(*req.OriginalURL) {
			return nil, ecode.CodeOriginalURLInvalid, ecode.Message(ecode.CodeOriginalURLInvalid)
		}
		updates["original_url"] = *req.OriginalURL
	}

	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if req.Password != nil {
		if *req.Password == "" {
			updates["password"] = ""
		} else {
			hashedPassword, err := utils.HashPassword(*req.Password)
			if err != nil {
				return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
			}
			updates["password"] = hashedPassword
		}
	}

	// 传 null 表示清除到期时间
	if req.ExpireAt == nil {
		updates["expire_at"] = nil
	}

	if req.ExpireAt != nil {
		expireTime := time.Unix(*req.ExpireAt, 0)
		if expireTime.Before(time.Now()) {
			return nil, ecode.CodeExpireAtInvalid, ecode.Message(ecode.CodeExpireAtInvalid)
		}
		updates["expire_at"] = expireTime
	}

	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return nil, ecode.CodeStatusInvalid, ecode.Message(ecode.CodeStatusInvalid)
		}
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		return nil, ecode.CodeNothingToUpdate, ecode.Message(ecode.CodeNothingToUpdate)
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.linkRepo.Update(tx, shortLink, updates)
	}); err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	updatedLink, err := s.linkRepo.GetOwnedByShortCode(userID, shortCode)
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	_ = s.linkRepo.DeleteCachedShortLink(shortCode)
	go s.linkRepo.SetCachedShortLink(*updatedLink)

	return buildShortLinkDetailResponse(*updatedLink, baseURL), ecode.CodeOK, "更新成功"
}

func (s *LinkService) DeleteShortLink(userID uint64, shortCode string) (int, string) {
	shortCode = strings.TrimSpace(shortCode)
	if shortCode == "" {
		return ecode.CodeShortCodeEmpty, ecode.Message(ecode.CodeShortCodeEmpty)
	}

	shortLink, err := s.linkRepo.GetOwnedByShortCode(userID, shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ecode.CodeShortLinkNotFound, ecode.Message(ecode.CodeShortLinkNotFound)
		}
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.linkRepo.Delete(tx, shortLink)
	}); err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	_ = s.linkRepo.DeleteCachedShortLink(shortCode)
	return ecode.CodeOK, "删除成功"
}

func (s *LinkService) resolveShortLink(shortCode string) (*model.ShortLink, error) {
	if link, hitCache, err := s.linkRepo.GetCachedShortLink(shortCode); err == nil && hitCache {
		return link, nil
	}

	link, err := s.linkRepo.GetByShortCode(shortCode)
	if err != nil {
		return nil, err
	}

	go s.linkRepo.SetCachedShortLink(*link)

	return link, nil
}

func buildShortLinkListItem(link model.ShortLink, baseURL string) dto.ShortLinkListItem {
	item := dto.ShortLinkListItem{
		ID:          link.ID,
		ShortCode:   link.ShortCode,
		ShortURL:    buildShortURL(baseURL, link.ShortCode),
		OriginalURL: link.OriginalURL,
		Remark:      link.Remark,
		Status:      link.Status,
		HasPassword: link.Password != "",
		ClickCount:  link.ClickCount,
		CreatedAt:   link.CreatedAt.Unix(),
		UpdatedAt:   link.UpdatedAt.Unix(),
	}

	if link.ExpireAt != nil {
		expireAt := link.ExpireAt.Unix()
		item.ExpireAt = &expireAt
		item.IsExpired = time.Now().After(*link.ExpireAt)
	}

	return item
}

func buildShortLinkDetailResponse(link model.ShortLink, baseURL string) *dto.ShortLinkDetailResponse {
	resp := &dto.ShortLinkDetailResponse{
		ID:          link.ID,
		ShortCode:   link.ShortCode,
		ShortURL:    buildShortURL(baseURL, link.ShortCode),
		OriginalURL: link.OriginalURL,
		Remark:      link.Remark,
		Status:      link.Status,
		HasPassword: link.Password != "",
		ClickCount:  link.ClickCount,
		CreatedAt:   link.CreatedAt.Unix(),
		UpdatedAt:   link.UpdatedAt.Unix(),
	}

	if link.ExpireAt != nil {
		expireAt := link.ExpireAt.Unix()
		resp.ExpireAt = &expireAt
		resp.IsExpired = time.Now().After(*link.ExpireAt)
	}

	return resp
}

func buildShortURL(baseURL, shortCode string) string {
	return fmt.Sprintf("%s/%s", normalizeBaseURL(baseURL), shortCode)
}

func (s *LinkService) generateUniqueShortCode() (string, error) {
	for i := 0; i < 3; i++ {
		shortCode, err := generateShortCode()
		if err != nil {
			return "", err
		}

		exists, err := s.linkRepo.ExistsByShortCode(shortCode)
		if err != nil {
			return "", err
		}
		if !exists {
			return shortCode, nil
		}
	}

	return "", errors.New("short code collision")
}

func generateShortCode() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	code := make([]byte, length)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range code {
		code[i] = chars[r.Intn(len(chars))]
	}

	return string(code), nil
}

func normalizeBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
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
