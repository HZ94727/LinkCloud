package controller

import "github.com/gin-gonic/gin"

func Redirect(c *gin.Context)        {} // 公开跳转
func CreateShortLink(c *gin.Context) {}
func GetShortLinks(c *gin.Context)   {}
func GetShortLink(c *gin.Context)    {}
func UpdateShortLink(c *gin.Context) {}
func DeleteShortLink(c *gin.Context) {}
