# LinkCloud 错误码说明

本文档用于同步当前后端返回的 `code` 与 `message` 约定，避免接口行为和前端展示脱节。

## 返回格式

大部分接口统一返回：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

其中：

- `code = 0` 表示成功
- `message` 是给前端/用户看的提示文案
- `data` 在成功时返回业务数据

## 通用错误码

| code | 含义 | 说明 |
| --- | --- | --- |
| 1001 | `CodeInvalidParam` | 请求参数不完整或格式不正确 |
| 1002 | `CodeUnauthorized` | 未登录或登录已过期 |
| 1003 | `CodeForbidden` | 无权限 |
| 1004 | `CodeNotFound` | 通用“未找到”错误，部分场景会被业务层复用为自定义文案 |
| 1005 | `CodeNothingToUpdate` | 未检测到需要更新的内容 |

## 认证 / 用户

| code | 含义 | 说明 |
| --- | --- | --- |
| 1101 | `CodeCaptchaNotFound` | 验证码不存在或已过期 |
| 1102 | `CodeCaptchaInvalid` | 验证码不正确 |
| 1103 | `CodeUserNotFound` | 用户不存在 |
| 1104 | `CodeUserNameOrPasswordBad` | 用户名或密码错误 |
| 1105 | `CodeEmailAlreadyRegistered` | 邮箱已被注册 |
| 1106 | `CodeUserNameAlreadyUsed` | 用户名已被使用 |
| 1107 | `CodeUserNameEmpty` | 用户名不能为空 |
| 1108 | `CodeUserNameLengthInvalid` | 用户名长度需为 3-20 个字符 |
| 1109 | `CodePasswordEmpty` | 密码不能为空 |
| 1110 | `CodePasswordLengthInvalid` | 密码长度需为 6-20 个字符 |
| 1111 | `CodeOldPasswordRequired` | 修改密码需要提供旧密码 |
| 1112 | `CodeNewPasswordEmpty` | 新密码不能为空 |
| 1113 | `CodeNewPasswordLengthInvalid` | 新密码长度需为 6-20 个字符 |
| 1114 | `CodeOldPasswordInvalid` | 旧密码错误 |
| 1115 | `CodeNeedRelogin` | 用户信息修改成功，请重新登录 |
| 1116 | `CodeCaptchaSendTooFrequent` | 操作太频繁，请 60 秒后再试 |
| 1117 | `CodeLoginLocked` | 密码错误次数过多，请 15 分钟后重试 |
| 1118 | `CodeResetLinkExists` | 已存在未过期的重置链接，请先查看邮件 |

## 短链接

| code | 含义 | 说明 |
| --- | --- | --- |
| 1201 | `CodeOriginalURLInvalid` | 原始链接格式不正确 |
| 1202 | `CodeQuotaInsufficient` | 配额不足，请充值 |
| 1203 | `CodeShortCodeEmpty` | 短码不能为空 |
| 1204 | `CodeShortLinkNotFound` | 短链接不存在 |
| 1205 | `CodeShortLinkExpired` | 短链接已过期 |
| 1206 | `CodeShortLinkDisabled` | 短链接已被禁用 |
| 1207 | `CodeShortLinkNeedPassword` | 该链接需要密码访问 |
| 1208 | `CodeShortLinkPasswordBad` | 密码错误 |
| 1209 | `CodeExpireAtInvalid` | 过期时间不能早于当前时间 |
| 1210 | `CodeStatusInvalid` | 状态值无效，只能为 0 或 1 |
| 1211 | `CodeShortLinkPasswordLock` | 密码错误次数过多，请 5 分钟后重试 |

## 统计 / 日志

| code | 含义 | 说明 |
| --- | --- | --- |
| 1301 | `CodeTimeRangeInvalid` | 时间范围无效 |

## 系统

| code | 含义 | 说明 |
| --- | --- | --- |
| 1901 | `CodeSystemBusy` | 系统繁忙，请稍后再试 |
| 1902 | `CodeTooManyRequests` | 请求过于频繁，请稍后再试 |

## 重置密码链路说明

### 公开页面

- `GET /reset-password`
- 通过静态文件返回重置密码页面

### 链接校验

- `GET /api/v1/auth/reset/validate?token=xxx`
- 页面加载时先调用该接口校验 token 是否仍然有效

### 提交重置

- `POST /api/v1/auth/reset`
- 成功后会更新用户密码，并删除 Redis 中的重置 token 以及有效链接标记

### 业务约束

- 同一用户如果已经存在未过期的重置链接，再次请求 `forgot` 会返回 `CodeResetLinkExists`
- 重置成功后，原 token 只能使用一次

## 需要留意的约定

- `CodeNotFound` 的默认文案是“页面不存在”，但在重置链接场景里，业务层会返回自定义文案“重置链接无效或已过期”
- `CodeResetLinkExists` 是新增码，前端如果做了错误码分支处理，需要补上这一项
- `AuthMiddleware` 目前返回的是 HTTP 401 的固定 JSON，不走 `ecode` 体系
