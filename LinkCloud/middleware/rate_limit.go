package middleware

import (
	"net/http"
	"strconv"

	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/repository"
	"github.com/gin-gonic/gin"
)

func TokenBucketRateLimit(scope string, capacity int64, refillPerSecond float64) gin.HandlerFunc {
	securityRepo := repository.NewSecurityRepository()

	return func(c *gin.Context) {
		allowed, retryMS, err := securityRepo.AllowTokenBucket(scope, c.ClientIP(), capacity, refillPerSecond)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    ecode.CodeSystemBusy,
				"message": ecode.Message(ecode.CodeSystemBusy),
			})
			c.Abort()
			return
		}

		if !allowed {
			if retryMS > 0 {
				c.Header("Retry-After", strconv.FormatInt((retryMS+999)/1000, 10))
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    ecode.CodeTooManyRequests,
				"message": ecode.Message(ecode.CodeTooManyRequests),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
