package utils

import (
	"fmt"
	"log"

	"github.com/wneessen/go-mail"
)

var (
	smtpHost  = "smtp.163.com"
	smtpPort  = 465
	fromEmail = "gyc94727@163.com"
	authCode  = "TJsuW38GCzKeMQjw"
)

// SendEmail 通用邮件发送函数
func SendEmail(to, subject, body string) error {
	// 创建邮件消息
	m := mail.NewMsg()
	if err := m.From(fromEmail); err != nil {
		return fmt.Errorf("设置发件人失败: %v", err)
	}
	if err := m.To(to); err != nil {
		return fmt.Errorf("设置收件人失败: %v", err)
	}
	m.Subject(subject)
	m.SetBodyString(mail.TypeTextHTML, body)

	// 创建 SMTP 客户端
	c, err := mail.NewClient(smtpHost,
		mail.WithPort(smtpPort),
		mail.WithSSL(),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(fromEmail),
		mail.WithPassword(authCode),
	)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %v", err)
	}

	// 发送
	if err := c.DialAndSend(m); err != nil {
		return fmt.Errorf("发送失败: %v", err)
	}

	log.Printf("邮件已发送到: %s", to)
	return nil
}

// SendVerificationCode 发送验证码专用函数
func SendVerificationCode(email, code string) error {
	subject := "【LinkCloud】邮箱验证码"

	body := fmt.Sprintf(`
        <!DOCTYPE html>
        <html>
        <head><meta charset="UTF-8"></head>
        <body style="font-family: Arial, sans-serif;">
            <div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
                <h2 style="color: #1a73e8;">LinkCloud</h2>
                <p>您好，</p>
                <p>您正在注册 LinkCloud 账号，验证码是：</p>
                <p style="font-size: 32px; font-weight: bold; color: #1a73e8; letter-spacing: 4px;">%s</p>
                <p style="color: #666;">验证码 5 分钟内有效，请勿泄露给他人。</p>
                <p style="color: #999; font-size: 12px;">如果不是您本人操作，请忽略此邮件。</p>
                <hr style="border: none; border-top: 1px solid #e0e0e0;">
                <p style="color: #999; font-size: 12px;">© 2026 LinkCloud 短链云</p>
            </div>
        </body>
        </html>
    `, code)

	return SendEmail(email, subject, body)
}
