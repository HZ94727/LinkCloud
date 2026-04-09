package ecode

const (
	CodeOK = 0

	// 1000-1099 通用错误
	CodeInvalidParam    = 1001
	CodeUnauthorized    = 1002
	CodeForbidden       = 1003
	CodeNotFound        = 1004
	CodeNothingToUpdate = 1005

	// 1100-1199 认证/用户
	CodeCaptchaSendFailed        = 1101
	CodeCaptchaExpired           = 1102
	CodeCaptchaInvalid           = 1103
	CodeUserNotFound             = 1104
	CodeUserNameOrPasswordBad    = 1105
	CodeEmailAlreadyRegistered   = 1106
	CodeUserNameAlreadyUsed      = 1107
	CodeUserNameEmpty            = 1108
	CodeUserNameLengthInvalid    = 1109
	CodePasswordEmpty            = 1110
	CodePasswordLengthInvalid    = 1111
	CodeOldPasswordRequired      = 1112
	CodeNewPasswordEmpty         = 1113
	CodeNewPasswordLengthInvalid = 1114
	CodeOldPasswordInvalid       = 1115
	CodeNeedRelogin              = 1116
	CodeUserUpdateFail           = 1117

	// 1200-1299 短链接
	CodeOriginalURLInvalid    = 1201
	CodeQuotaInsufficient     = 1202
	CodeShortCodeGenerateFail = 1203
	CodeShortLinkCreateFail   = 1204
	CodeShortCodeEmpty        = 1205
	CodeShortLinkNotFound     = 1206
	CodeShortLinkExpired      = 1207
	CodeShortLinkDisabled     = 1208
	CodeShortLinkNeedPassword = 1209
	CodeShortLinkPasswordBad  = 1210
	CodeExpireAtInvalid       = 1211
	CodeStatusInvalid         = 1212
	CodeShortLinkUpdateFail   = 1213
	CodeShortLinkDeleteFail   = 1214

	// 1300-1399 统计/日志
	CodeStatsQueryFailed = 1301
	CodeLogQueryFailed   = 1302
	CodeTimeRangeInvalid = 1303

	// 1900-1999 系统
	CodeSystemBusy = 1900
)

var messages = map[int]string{
	CodeOK: "ok",

	CodeInvalidParam:    "请求参数不完整或格式不正确",
	CodeUnauthorized:    "未登录或登录已过期",
	CodeForbidden:       "无权限",
	CodeNotFound:        "资源不存在",
	CodeNothingToUpdate: "未检测到需要更新的内容",

	CodeCaptchaSendFailed:        "验证码发送失败, 请稍后再试",
	CodeCaptchaExpired:           "验证码已过期, 请重新获取",
	CodeCaptchaInvalid:           "验证码不正确",
	CodeUserNotFound:             "用户不存在",
	CodeUserNameOrPasswordBad:    "用户名或密码错误",
	CodeEmailAlreadyRegistered:   "邮箱已被注册",
	CodeUserNameAlreadyUsed:      "用户名已被使用",
	CodeUserNameEmpty:            "用户名不能为空",
	CodeUserNameLengthInvalid:    "用户名长度需为3-20个字符",
	CodePasswordEmpty:            "密码不能为空",
	CodePasswordLengthInvalid:    "密码长度需为6-20个字符",
	CodeOldPasswordRequired:      "修改密码需要提供旧密码",
	CodeNewPasswordEmpty:         "新密码不能为空",
	CodeNewPasswordLengthInvalid: "新密码长度需为6-20个字符",
	CodeOldPasswordInvalid:       "旧密码错误",
	CodeNeedRelogin:              "用户信息修改成功，请重新登录",
	CodeUserUpdateFail:           "更新失败",

	CodeOriginalURLInvalid:    "原始链接格式不正确",
	CodeQuotaInsufficient:     "配额不足, 请充值",
	CodeShortCodeGenerateFail: "短码生成失败, 请重试",
	CodeShortLinkCreateFail:   "生成短链接失败, 请稍后再试",
	CodeShortCodeEmpty:        "短码不能为空",
	CodeShortLinkNotFound:     "短链接不存在",
	CodeShortLinkExpired:      "短链接已过期",
	CodeShortLinkDisabled:     "短链接已被禁用",
	CodeShortLinkNeedPassword: "该链接需要密码访问",
	CodeShortLinkPasswordBad:  "密码错误",
	CodeExpireAtInvalid:       "过期时间不能早于当前时间",
	CodeStatusInvalid:         "状态值无效, 只能为0或1",
	CodeShortLinkUpdateFail:   "更新失败",
	CodeShortLinkDeleteFail:   "删除失败",

	CodeStatsQueryFailed: "统计查询失败",
	CodeLogQueryFailed:   "查询访问日志失败",
	CodeTimeRangeInvalid: "时间范围无效",

	CodeSystemBusy: "系统繁忙, 请稍后再试",
}

func Message(code int) string {
	if msg, ok := messages[code]; ok {
		return msg
	}
	return "未知错误"
}
