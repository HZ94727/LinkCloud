package controller

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/model"
	"gitea.com/hz/linkcloud/utils"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func Login(c *gin.Context) {
	var req struct {
		UserName string `json:"user_name"`
		Password string `json:"password"`
	}
	c.BindJSON(&req)

	// 查用户
	var user model.User
	result := database.DB.Where("user_name = ?", req.UserName).First(&user)
	if result.Error != nil {
		c.JSON(200, gin.H{"code": -3, "message": "用户不存在"})
		return
	}

	// 验证密码
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(200, gin.H{"code": -4, "message": "用户名或密码错误"})
		return
	}

	// 生成token
	token, _ := utils.GenerateToken(user.ID, user.UserName)

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"id":              user.ID,
			"user_name":       user.UserName,
			"email":           user.Email,
			"quota":           user.Quota,
			"used_quota":      user.UsedQuota,
			"remaining_quota": user.Quota - user.UsedQuota,
			"token":           token,
			"created_at":      user.CreatedAt.Unix(),
		},
	})
}

// 发送验证码
func SendCaptcha(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    -1,
			"message": "邮箱不能为空或格式不正确",
		})
		return
	}

	email := req.Email

	// 3. 生成6位随机验证码
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	fmt.Println("code is: ", code)

	codeKey := fmt.Sprintf("captcha:%s", email)
	database.Redis.Set(database.Ctx, codeKey, code, 5*time.Minute)

	if err := utils.SendVerificationCode(email, code); err != nil {
		c.JSON(200, gin.H{
			"code":    -2,
			"message": "验证码发送失败, 请稍后再试",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "验证码已发送到邮箱, 请注意查收",
	})
}

// controller/auth.go
func Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		UserName string `json:"user_name" binding:"required,min=3,max=20"`
		Password string `json:"password" binding:"required,min=6,max=20"`
		Captcha  string `json:"captcha" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    -1,
			"message": "邮箱/用户名/密码/验证码格式不正确",
		})
		return
	}

	// 1. 验证验证码
	codeKey := fmt.Sprintf("captcha:%s", req.Email)
	storedCode, err := database.Redis.Get(database.Ctx, codeKey).Result()

	// 键不存在的情况下，err == redis.Nil
	// 键存在的情况下，err == nil

	// fmt.Println(err == nil, err == redis.Nil, len(storedCode))

	if err != nil {
		// 有错误，分两种情况
		if err == redis.Nil {
			// 键不存在 = 验证码已过期
			c.JSON(200, gin.H{
				"code":    -2,
				"message": "验证码已过期, 请重新获取",
			})
		} else {
			// 其他错误（Redis连接失败等）
			c.JSON(200, gin.H{
				"code":    -3,
				"message": "系统繁忙, 请稍后再试",
			})
		}
		return
	}

	if storedCode != req.Captcha {
		c.JSON(200, gin.H{
			"code":    -4,
			"message": "验证码不正确",
		})
		return
	}

	// 2. 验证通过后立即删除验证码（防止重复使用）
	database.Redis.Del(database.Ctx, codeKey)

	// 3. 检查邮箱是否已存在
	var existUser model.User
	if err := database.DB.Where("email = ?", req.Email).First(&existUser).Error; err == nil {
		c.JSON(200, gin.H{
			"code":    -5,
			"message": "邮箱已被注册",
		})
		return
	}

	// 4. 检查用户名是否已存在
	if err := database.DB.Where("user_name = ?", req.UserName).First(&existUser).Error; err == nil {
		c.JSON(200, gin.H{
			"code":    -6,
			"message": "用户名已被使用",
		})
		return
	}

	// 5. 加密密码
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    -7,
			"message": "注册失败, 请稍后再试",
		})
		return
	}

	// 6. 创建用户
	user := model.User{
		Email:     req.Email,
		UserName:  req.UserName,
		Password:  hashedPassword,
		Status:    1,
		Quota:     100, // 默认配额
		UsedQuota: 0,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(200, gin.H{
			"code":    -7,
			"message": "注册失败, 请稍后再试",
		})
		return
	}

	// 8. 返回成功响应
	c.JSON(200, gin.H{
		"code":    0,
		"message": "注册成功",
		"data": gin.H{
			"id":              user.ID,
			"email":           user.Email,
			"user_name":       user.UserName,
			"created_at":      user.CreatedAt.Unix(),
			"quota":           user.Quota,
			"used_quota":      user.UsedQuota,
			"remaining_quota": user.Quota - user.UsedQuota,
		},
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
