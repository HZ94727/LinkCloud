package controller

import (
	"net/http"

	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/service"
	"github.com/gin-gonic/gin"
)

func GetUserInfo(c *gin.Context) {
	userService := service.DefaultUserService()
	resp, code, message := userService.GetUserInfo(currentUserID(c))
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

func UpdateUserInfo(c *gin.Context) {
	var req dto.UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    ecode.CodeInvalidParam,
			"message": ecode.Message(ecode.CodeInvalidParam),
		})
		return
	}

	userService := service.DefaultUserService()
	resp, code, message := userService.UpdateUserInfo(currentUserID(c), req)
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
