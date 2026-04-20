package controller

import (
	"net/http"
	"net/url"

	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/service"
	"github.com/gin-gonic/gin"
)

func Redirect(c *gin.Context) {
	shortCode := c.Param("short_code")
	password := c.Query("password")

	link, code, message := service.DefaultLinkService().ResolveShortLink(
		shortCode,
		password,
		c.ClientIP(),
		c.Request.UserAgent(),
		c.Request.Referer(),
	)
	if code != ecode.CodeOK {
		if code == ecode.CodeShortLinkNeedPassword {
			c.Redirect(http.StatusFound, buildShortLinkPasswordPageURL(shortCode, ""))
			return
		}

		if code == ecode.CodeShortLinkPasswordBad || code == ecode.CodeShortLinkPasswordLock {
			c.Redirect(http.StatusFound, buildShortLinkPasswordPageURL(shortCode, message))
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.Redirect(http.StatusFound, link.OriginalURL)
}

func buildShortLinkPasswordPageURL(shortCode, message string) string {
	redirectURL := "/short-link-password?short_code=" + url.QueryEscape(shortCode)
	if message != "" {
		redirectURL += "&message=" + url.QueryEscape(message)
	}
	return redirectURL
}

func CreateShortLink(c *gin.Context) {
	var req dto.CreateShortLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	linkService := service.DefaultLinkService()
	resp, code, message := linkService.CreateShortLink(currentUserID(c), req)
	if code != ecode.CodeOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
		"data":    resp,
	})
}

func GetShortLinks(c *gin.Context) {
	var query dto.ShortLinkListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	keywordMap := c.QueryMap("keyword")
	linkService := service.DefaultLinkService()
	resp, code, message := linkService.ListShortLinks(currentUserID(c), dto.ShortLinkListRequest{
		Page:       query.Page,
		PageSize:   query.PageSize,
		SortBy:     query.SortBy,
		SortOrder:  query.SortOrder,
		Status:     query.Status,
		Keywords:   keywordMap,
		FuzzyQuery: query.FuzzyQuery,
	}, buildBaseURL(c))

	if code != ecode.CodeOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
		"data":    resp,
	})
}

func GetShortLink(c *gin.Context) {
	shortCode := c.Param("short_code")

	linkService := service.DefaultLinkService()
	resp, code, message := linkService.GetShortLinkDetail(currentUserID(c), shortCode, buildBaseURL(c))
	if code != ecode.CodeOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
		"data":    resp,
	})
}

func UpdateShortLink(c *gin.Context) {
	shortCode := c.Param("short_code")
	var req dto.UpdateShortLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	linkService := service.DefaultLinkService()
	resp, code, message := linkService.UpdateShortLink(currentUserID(c), shortCode, req, buildBaseURL(c))
	if code != ecode.CodeOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
		"data":    resp,
	})
}

func DeleteShortLink(c *gin.Context) {
	shortCode := c.Param("short_code")

	linkService := service.DefaultLinkService()

	code, message := linkService.DeleteShortLink(currentUserID(c), shortCode)
	if code != ecode.CodeOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
	})
}

func buildBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

func currentUserID(c *gin.Context) uint64 {
	if value, exists := c.Get("user_id"); exists {
		if userID, ok := value.(uint64); ok {
			return userID
		}
	}
	return 0
}
