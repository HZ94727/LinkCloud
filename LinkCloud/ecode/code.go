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
	CodeCaptchaNotFound          = 1101
	CodeCaptchaInvalid           = 1102
	CodeUserNotFound             = 1103
	CodeUserNameOrPasswordBad    = 1104
	CodeEmailAlreadyRegistered   = 1105
	CodeUserNameAlreadyUsed      = 1106
	CodeUserNameEmpty            = 1107
	CodeUserNameLengthInvalid    = 1108
	CodePasswordEmpty            = 1109
	CodePasswordLengthInvalid    = 1110
	CodeOldPasswordRequired      = 1111
	CodeNewPasswordEmpty         = 1112
	CodeNewPasswordLengthInvalid = 1113
	CodeOldPasswordInvalid       = 1114
	CodeNeedRelogin              = 1115
	CodeCaptchaSendTooFrequent   = 1116
	CodeLoginLocked              = 1117

	// 1200-1299 短链接
	CodeOriginalURLInvalid    = 1201
	CodeQuotaInsufficient     = 1202
	CodeShortCodeEmpty        = 1203
	CodeShortLinkNotFound     = 1204
	CodeShortLinkExpired      = 1205
	CodeShortLinkDisabled     = 1206
	CodeShortLinkNeedPassword = 1207
	CodeShortLinkPasswordBad  = 1208
	CodeExpireAtInvalid       = 1209
	CodeStatusInvalid         = 1210
	CodeShortLinkPasswordLock = 1211

	// 1300-1399 统计/日志
	CodeTimeRangeInvalid = 1301

	// 1900-1999 系统
	CodeSystemBusy      = 1901
	CodeTooManyRequests = 1902
)

var messages = map[int]string{
	CodeOK: "ok",

	CodeInvalidParam:    "请求参数不完整或格式不正确",
	CodeUnauthorized:    "未登录或登录已过期",
	CodeForbidden:       "无权限",
	CodeNotFound:        "页面不存在",
	CodeNothingToUpdate: "未检测到需要更新的内容",

	CodeCaptchaNotFound:          "验证码不存在或已过期, 请重新获取",
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
	CodeCaptchaSendTooFrequent:   "操作太频繁, 请60秒后再试",
	CodeLoginLocked:              "密码错误次数过多, 请15分钟后重试",

	CodeOriginalURLInvalid:    "原始链接格式不正确",
	CodeQuotaInsufficient:     "配额不足, 请充值",
	CodeShortCodeEmpty:        "短码不能为空",
	CodeShortLinkNotFound:     "短链接不存在",
	CodeShortLinkExpired:      "短链接已过期",
	CodeShortLinkDisabled:     "短链接已被禁用",
	CodeShortLinkNeedPassword: "该链接需要密码访问",
	CodeShortLinkPasswordBad:  "密码错误",
	CodeExpireAtInvalid:       "过期时间不能早于当前时间",
	CodeStatusInvalid:         "状态值无效, 只能为0或1",
	CodeShortLinkPasswordLock: "密码错误次数过多, 请5分钟后重试",

	CodeTimeRangeInvalid: "时间范围无效",

	CodeSystemBusy:      "系统繁忙, 请稍后再试",
	CodeTooManyRequests: "请求过于频繁, 请稍后再试",
}

func Message(code int) string {
	if msg, ok := messages[code]; ok {
		return msg
	}
	return "未知错误"
}
