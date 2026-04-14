package routes

import (
	"gitea.com/hz/linkcloud/controller"
	"gitea.com/hz/linkcloud/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	// 静态文件
	r.StaticFile("/reset-password", "./templates/reset_password.html")
	r.StaticFile("/favicon.ico", "./templates/favicon.ico")

	// 公开接口
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", controller.Login)
		auth.POST("/register", middleware.TokenBucketRateLimit("auth-register", 10, 5), controller.Register)
		auth.POST("/captcha", middleware.TokenBucketRateLimit("auth-captcha", 5, 1), controller.SendCaptcha)
		auth.POST("/forgot", middleware.TokenBucketRateLimit("auth-forgot", 6, 3), controller.ForgotPassword)
		auth.POST("/reset", middleware.TokenBucketRateLimit("auth-reset", 6, 3), controller.ResetPassword)
	}

	r.GET("/:short_code", middleware.TokenBucketRateLimit("short-link", 50, 25), controller.Redirect)

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
