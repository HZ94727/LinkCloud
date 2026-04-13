package service

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/model"
	"gitea.com/hz/linkcloud/repository"
	"gitea.com/hz/linkcloud/utils"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	captchaRepo *repository.CaptchaRepository
}

func NewAuthService(userRepo *repository.UserRepository, captchaRepo *repository.CaptchaRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		captchaRepo: captchaRepo,
	}
}

func DefaultAuthService() *AuthService {
	return NewAuthService(repository.NewUserRepository(), repository.NewCaptchaRepository())
}

func (s *AuthService) SendCaptcha(req dto.SendCaptchaRequest) (int, string) {
	// 随机生成6位数字验证码
	captcha := fmt.Sprintf("%06d", rand.Intn(1000000))
	if err := utils.SendCaptcha(req.Email, captcha); err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	// 写入 Redis 缓存，5分钟有效
	if err := s.captchaRepo.Set(req.Email, captcha, 5*time.Minute); err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return ecode.CodeOK, "验证码已发送到邮箱, 请注意查收"
}

func (s *AuthService) Login(req dto.LoginRequest) (*dto.LoginResponse, int, string) {
	user, err := s.userRepo.GetByUserName(req.UserName)
	if err != nil {
		return nil, ecode.CodeUserNameOrPasswordBad, ecode.Message(ecode.CodeUserNameOrPasswordBad)
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, ecode.CodeUserNameOrPasswordBad, ecode.Message(ecode.CodeUserNameOrPasswordBad)
	}

	token, err := utils.GenerateToken(user.ID, user.UserName)
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return &dto.LoginResponse{
		ID:             user.ID,
		UserName:       user.UserName,
		Email:          user.Email,
		Quota:          user.Quota,
		UsedQuota:      user.UsedQuota,
		RemainingQuota: user.Quota - user.UsedQuota,
		Token:          token,
		CreatedAt:      user.CreatedAt.Unix(),
	}, ecode.CodeOK, "登录成功"
}

func (s *AuthService) Register(req dto.RegisterRequest) (*dto.RegisterResponse, int, string) {
	// 检查用户名和密码长度
	if !utils.IsValidUserNameLength(req.UserName) {
		return nil, ecode.CodeUserNameLengthInvalid, ecode.Message(ecode.CodeUserNameLengthInvalid)
	}
	if !utils.IsValidPasswordLength(req.Password) {
		return nil, ecode.CodePasswordLengthInvalid, ecode.Message(ecode.CodePasswordLengthInvalid)
	}

	// 获取 Redis 存储的验证码
	storedCode, err := s.captchaRepo.Get(req.Email)

	log.Println("stortCode and err is: ", storedCode, err)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ecode.CodeCaptchaNotFound, ecode.Message(ecode.CodeCaptchaNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	// 验证码不匹配
	if storedCode != req.Captcha {
		return nil, ecode.CodeCaptchaInvalid, ecode.Message(ecode.CodeCaptchaInvalid)
	}

	// 验证成功就立即删除验证码
	if err := s.captchaRepo.Delete(req.Email); err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	// 尝试查找邮箱, 且找到
	emailExists, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	if emailExists {
		return nil, ecode.CodeEmailAlreadyRegistered, ecode.Message(ecode.CodeEmailAlreadyRegistered)
	}

	// 判断用户是否存在
	userExists, err := s.userRepo.ExistsByUserName(req.UserName)
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	if userExists {
		return nil, ecode.CodeUserNameAlreadyUsed, ecode.Message(ecode.CodeUserNameAlreadyUsed)
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	user := &model.User{
		Email:     req.Email,
		UserName:  req.UserName,
		Password:  hashedPassword,
		Status:    1,
		Quota:     100,
		UsedQuota: 0,
	}

	// 插入数据库
	if err := s.userRepo.Create(user); err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return &dto.RegisterResponse{
		ID:             user.ID,
		Email:          user.Email,
		UserName:       user.UserName,
		CreatedAt:      user.CreatedAt.Unix(),
		Quota:          user.Quota,
		UsedQuota:      user.UsedQuota,
		RemainingQuota: user.Quota - user.UsedQuota,
	}, ecode.CodeOK, "注册成功"
}

func (s *AuthService) ForgotPassword(req dto.ForgotPasswordRequest) (int, string) {
	user, err := s.userRepo.GetByEmailAndUserName(req.Email, req.UserName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ecode.CodeNotFound, "邮箱或用户名不匹配"
	}
	fmt.Println(user, err)

	if err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	resetToken, err := utils.GenerateResetToken()
	if err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	key := fmt.Sprintf("reset:%s", resetToken)
	err = database.Redis.Set(database.Ctx, key, user.ID, 1*time.Hour).Err()
	if err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	// baseURL := "http://192.168.10.27:8080/api/v1/auth/reset"
	baseURL := "http://192.168.10.27:8080/reset-password" // 改成你的后端地址
	resetURL := fmt.Sprintf("%s?token=%s", baseURL, resetToken)
	if err := utils.SendResetLink(req.Email, resetURL); err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return ecode.CodeOK, "重置链接已发送到您的邮箱, 请注意查收"
}

func (s *AuthService) ResetPassword(req dto.ResetPasswordRequest) (int, string) {
	key := fmt.Sprintf("%s:%s", "reset", req.Token)
	id, err := database.Redis.Get(database.Ctx, key).Result()

	if err == redis.Nil {
		return ecode.CodeNotFound, "重置链接无效或已过期"
	}

	if err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	user := &model.User{
		ID: userID,
	}

	newPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	updates := make(map[string]any)
	updates["password"] = newPassword

	if err := s.userRepo.Update(database.DB, user, updates); err != nil {
		return ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}
	return ecode.CodeOK, "重置密码成功"
}
