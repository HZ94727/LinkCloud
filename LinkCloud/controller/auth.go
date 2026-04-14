package controller

import (
	"log"
	"net/http"
	"strconv"

	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/repository"
	"gitea.com/hz/linkcloud/service"
	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	securityRepo := repository.NewSecurityRepository()
	allowed, retryMS, err := securityRepo.AllowTokenBucketByKey("auth-login", c.ClientIP()+":"+req.UserName, 10, 5)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    ecode.CodeSystemBusy,
			"message": ecode.Message(ecode.CodeSystemBusy),
		})
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
		return
	}

	authService := service.DefaultAuthService()
	resp, code, message := authService.Login(req, c.ClientIP())
	if code != 0 {
		c.JSON(200, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
		"data":    resp,
	})
}

// 发送验证码
func SendCaptcha(c *gin.Context) {
	var req dto.SendCaptchaRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": "邮箱不能为空或格式不正确",
		})
		return
	}

	authService := service.DefaultAuthService()
	code, message := authService.SendCaptcha(req)
	if code != ecode.CodeOK {
		c.JSON(200, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
	})
}

// 用户注册
func Register(c *gin.Context) {
	var req dto.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	authService := service.DefaultAuthService()
	resp, code, message := authService.Register(req)
	if code != ecode.CodeOK {
		c.JSON(200, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    ecode.CodeOK,
		"message": message,
		"data":    resp,
	})
}

func Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 可选：记录退出日志
	log.Printf("用户 %d 退出登录", userID)

	// 可选：清除 Redis 中的 refresh token（如果有）
	// database.Redis.Del(database.Ctx, fmt.Sprintf("refresh_token:%d", userID))

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "退出成功",
	})
}

func ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	authService := service.DefaultAuthService()
	code, message := authService.ForgotPassword(req)

	c.JSON(200, gin.H{
		"code":    code,
		"message": message,
	})
}

func ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	authService := service.DefaultAuthService()
	code, message := authService.ResetPassword(req)

	c.JSON(200, gin.H{
		"code":    code,
		"message": message,
	})
}
