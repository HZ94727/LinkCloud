package controller

import (
	"net/http"
	"strconv"
	"time"

	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/service"
	"github.com/gin-gonic/gin"
)

func GetStats(c *gin.Context) {
	shortCode := c.Param("short_code")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	statsService := service.DefaultStatsService()
	resp, code, message := statsService.GetStats(currentUserID(c), shortCode, days, time.Now())
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

func GetLogs(c *gin.Context) {
	shortCode := c.Param("short_code")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	statsService := service.DefaultStatsService()
	resp, code, message := statsService.GetLogs(
		currentUserID(c),
		shortCode,
		page,
		size,
		c.Query("start_at"),
		c.Query("end_at"),
	)
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
