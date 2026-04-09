package controller

import (
	"fmt"
	"net/http"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GetUserInfo 获取当前登录用户信息
func GetUserInfo(c *gin.Context) {
	// 1. 从中间件获取用户ID
	userID, exists := c.Get("user_id")
	fmt.Println("user id is: ", userID)
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 2. 查询用户信息
	var user model.User
	result := database.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "用户不存在",
		})
		return
	}

	// 3. 返回用户信息（不包含敏感字段）
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": gin.H{
			"id":              user.ID,
			"user_name":       user.UserName,
			"email":           user.Email,
			"used_quota":      user.UsedQuota,
			"quota":           user.Quota,
			"created_at":      user.CreatedAt.Unix(),
			"updated_at":      user.UpdatedAt.Unix(),
			"remaining_quota": user.Quota - user.UsedQuota,
		},
	})
}

// UpdateUserInfo 更新当前登录用户信息（用户名/密码）
func UpdateUserInfo(c *gin.Context) {
	// 1. 从中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 2. 绑定请求体
	var req struct {
		UserName    *string `json:"user_name"`
		OldPassword *string `json:"old_password"`
		NewPassword *string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "参数错误",
		})
		return
	}

	// 3. 查询用户
	var user model.User
	result := database.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -3,
			"message": "用户不存在",
		})
		return
	}

	updates := make(map[string]interface{})
	needRelogin := false
	hasEffectiveChange := false

	// 4. 修改用户名
	if req.UserName != nil {
		if *req.UserName == "" {
			c.JSON(http.StatusOK, gin.H{
				"code":    -4,
				"message": "用户名不能为空",
			})
			return
		}
		if !isValidUserNameLength(*req.UserName) {
			c.JSON(http.StatusOK, gin.H{
				"code":    -5,
				"message": "用户名长度需为3-20个字符",
			})
			return
		}
		if *req.UserName != user.UserName {
			// 检查用户名是否已被占用
			var existUser model.User
			if err := database.DB.Where("user_name = ? AND id != ?", *req.UserName, userID).First(&existUser).Error; err == nil {
				c.JSON(http.StatusOK, gin.H{
					"code":    -6,
					"message": "用户名已被占用",
				})
				return
			}
			updates["user_name"] = *req.UserName
			hasEffectiveChange = true
			needRelogin = true // 修改用户名需要重新登录, 避免 JWT 中的 user_name 过期
		}
	}

	// 5. 修改密码
	if req.NewPassword != nil {
		// 必须提供旧密码
		if req.OldPassword == nil || *req.OldPassword == "" {
			c.JSON(http.StatusOK, gin.H{
				"code":    -7,
				"message": "修改密码需要提供旧密码",
			})
			return
		}

		// 新密码不能为空
		if *req.NewPassword == "" {
			c.JSON(http.StatusOK, gin.H{
				"code":    -8,
				"message": "新密码不能为空",
			})
			return
		}

		// 验证旧密码
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(*req.OldPassword)); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    -9,
				"message": "旧密码错误",
			})
			return
		}

		// 加密新密码
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    -10,
				"message": "系统繁忙, 请稍后再试",
			})
			return
		}
		updates["password"] = string(hashedPassword)
		hasEffectiveChange = true
		needRelogin = true // 修改密码需要重新登录
	}

	if !hasEffectiveChange {
		c.JSON(http.StatusOK, gin.H{
			"code":    -11,
			"message": "未检测到需要更新的内容",
		})
		return
	}

	// 6. 执行更新
	if len(updates) > 0 {
		if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    -12,
				"message": "更新失败",
			})
			return
		}
	}

	// 7. 查询更新后的用户信息
	var updatedUser model.User
	database.DB.Where("id = ?", userID).First(&updatedUser)

	// 8. 构建响应消息
	message := "更新成功"
	if needRelogin {
		message = "用户信息修改成功, 请重新登录"
	}

	// 9. 返回响应
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": message,
		"data": gin.H{
			"id":              updatedUser.ID,
			"user_name":       updatedUser.UserName,
			"email":           updatedUser.Email,
			"used_quota":      updatedUser.UsedQuota,
			"quota":           updatedUser.Quota,
			"created_at":      updatedUser.CreatedAt.Unix(),
			"updated_at":      updatedUser.UpdatedAt.Unix(),
			"need_relogin":    needRelogin, // 告诉前端是否需要重新登录
			"remaining_quota": updatedUser.Quota - updatedUser.UsedQuota,
		},
	})
}
