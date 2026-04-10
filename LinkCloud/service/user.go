package service

import (
	"errors"

	"gitea.com/hz/linkcloud/dto"
	"gitea.com/hz/linkcloud/ecode"
	"gitea.com/hz/linkcloud/model"
	"gitea.com/hz/linkcloud/repository"
	"gitea.com/hz/linkcloud/utils"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func DefaultUserService() *UserService {
	return NewUserService(repository.NewUserRepository())
}

func (s *UserService) GetUserInfo(userID uint64) (*dto.UserInfoResponse, int, string) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ecode.CodeUserNotFound, ecode.Message(ecode.CodeUserNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return buildUserInfoResponse(user), ecode.CodeOK, "获取成功"
}

func (s *UserService) UpdateUserInfo(userID uint64, req dto.UpdateUserInfoRequest) (*dto.UpdateUserInfoResponse, int, string) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ecode.CodeUserNotFound, ecode.Message(ecode.CodeUserNotFound)
		}
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	updates := make(map[string]any)
	needRelogin := false

	if req.UserName != nil {
		if *req.UserName == "" {
			return nil, ecode.CodeUserNameEmpty, ecode.Message(ecode.CodeUserNameEmpty)
		}
		if !utils.IsValidUserNameLength(*req.UserName) {
			return nil, ecode.CodeUserNameLengthInvalid, ecode.Message(ecode.CodeUserNameLengthInvalid)
		}
		if *req.UserName != user.UserName {
			if existUser, err := s.userRepo.GetByUserNameExceptID(*req.UserName, userID); err == nil && existUser != nil {
				return nil, ecode.CodeUserNameAlreadyUsed, ecode.Message(ecode.CodeUserNameAlreadyUsed)
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
			}
			updates["user_name"] = *req.UserName
			needRelogin = true
		}
	}

	if req.NewPassword != nil {
		if req.OldPassword == nil || *req.OldPassword == "" {
			return nil, ecode.CodeOldPasswordRequired, ecode.Message(ecode.CodeOldPasswordRequired)
		}
		if *req.NewPassword == "" {
			return nil, ecode.CodeNewPasswordEmpty, ecode.Message(ecode.CodeNewPasswordEmpty)
		}
		if !utils.IsValidPasswordLength(*req.NewPassword) {
			return nil, ecode.CodeNewPasswordLengthInvalid, ecode.Message(ecode.CodeNewPasswordLengthInvalid)
		}
		if !utils.CheckPasswordHash(*req.OldPassword, user.Password) {
			return nil, ecode.CodeOldPasswordInvalid, ecode.Message(ecode.CodeOldPasswordInvalid)
		}

		hashedPassword, err := utils.HashPassword(*req.NewPassword)
		if err != nil {
			return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
		}
		updates["password"] = hashedPassword
		needRelogin = true
	}

	if len(updates) == 0 {
		return nil, ecode.CodeNothingToUpdate, ecode.Message(ecode.CodeNothingToUpdate)
	}

	if err := s.userRepo.Update(nil, user, updates); err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	updatedUser, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, ecode.CodeSystemBusy, ecode.Message(ecode.CodeSystemBusy)
	}

	return buildUserUpdateResponse(updatedUser, needRelogin), ecode.CodeOK, mapNeedReloginMessage(needRelogin)
}

func buildUserInfoResponse(user *model.User) *dto.UserInfoResponse {
	return &dto.UserInfoResponse{
		ID:             user.ID,
		UserName:       user.UserName,
		Email:          user.Email,
		UsedQuota:      user.UsedQuota,
		Quota:          user.Quota,
		CreatedAt:      user.CreatedAt.Unix(),
		UpdatedAt:      user.UpdatedAt.Unix(),
		RemainingQuota: user.Quota - user.UsedQuota,
	}
}

func buildUserUpdateResponse(user *model.User, needRelogin bool) *dto.UpdateUserInfoResponse {
	return &dto.UpdateUserInfoResponse{
		ID:             user.ID,
		UserName:       user.UserName,
		Email:          user.Email,
		UsedQuota:      user.UsedQuota,
		Quota:          user.Quota,
		CreatedAt:      user.CreatedAt.Unix(),
		UpdatedAt:      user.UpdatedAt.Unix(),
		RemainingQuota: user.Quota - user.UsedQuota,
		NeedRelogin:    needRelogin,
	}
}

func mapNeedReloginMessage(needRelogin bool) string {
	if needRelogin {
		return ecode.Message(ecode.CodeNeedRelogin)
	}
	return "更新成功"
}
