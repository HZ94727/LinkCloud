package routes

import (
	"fmt"
	"net/http"
	"time"

	"gitea.com/hz/linkcloud/controller"
	"gitea.com/hz/linkcloud/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	// 静态文件
	r.StaticFile("/reset-password", "./templates/reset_password.html")
	r.StaticFile("/short-link-password", "./templates/short_link_password.html")
	r.StaticFile("/favicon.ico", "./templates/favicon.ico")

	// 公开接口
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", controller.Login)
		auth.POST("/register", middleware.TokenBucketRateLimit("auth-register", 10, 5), controller.Register)
		auth.POST("/captcha", middleware.TokenBucketRateLimit("auth-captcha", 5, 1), controller.SendCaptcha)
		auth.POST("/forgot", middleware.TokenBucketRateLimit("auth-forgot", 6, 3), controller.ForgotPassword)
		auth.GET("/reset/validate", controller.ValidateResetPasswordToken)
		auth.POST("/reset", middleware.TokenBucketRateLimit("auth-reset", 6, 3), controller.ResetPassword)
	}

	r.GET("/s/:short_code", middleware.TokenBucketRateLimit("short-link", 50, 25), controller.Redirect)

	r.GET("/test", func(ctx *gin.Context) {
		fmt.Println("/test router")
		go func() {
			time.Sleep(time.Second * 5)
			fmt.Println("request url is: ", ctx.Request.URL)
		}()
		ctx.String(http.StatusOK, "OK")
	})

	// 需要认证的接口
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		api.POST("/links", controller.CreateShortLink)
		api.GET("/links", controller.GetShortLinks)
		api.GET("/links/:short_code", controller.GetShortLink)
		api.PATCH("/links/:short_code", controller.UpdateShortLink)
		api.DELETE("/links/:short_code", controller.DeleteShortLink)
		api.GET("/stats/:short_code", controller.GetStats)
		api.GET("/stats/:short_code/logs", controller.GetLogs)
		api.GET("/user/info", controller.GetUserInfo)
		api.PATCH("/user/info", controller.UpdateUserInfo)
		api.POST("/auth/logout", controller.Logout)
	}

	return r
}
