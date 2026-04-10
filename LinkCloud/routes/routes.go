package routes

import (
	"gitea.com/hz/linkcloud/controller"
	"gitea.com/hz/linkcloud/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.StaticFile("/reset-password", "./templates/reset_password.html")
	// 公开接口
	r.POST("/api/v1/auth/login", controller.Login)
	r.POST("/api/v1/auth/register", controller.Register)
	r.POST("/api/v1/auth/captcha", controller.SendCaptcha)
	r.GET("/:short_code", controller.Redirect)
	r.POST("/api/v1/auth/forgot", controller.ForgotPassword)
	r.POST("/api/v1/auth/reset", controller.ResetPassword)

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
