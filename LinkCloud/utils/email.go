package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
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
func SendCaptcha(email, captcha string) error {
	tmpl, err := template.ParseFiles("templates/verification_code.html")
	if err != nil {
		return err
	}

	subject := "【LinkCloud】邮箱验证码"

	data := map[string]any{
		"Captcha": captcha,
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	return SendEmail(email, subject, body.String())
}

// SendResetLink 发送重置密码链接
func SendResetLink(email, resetURL string) error {
	tmpl, err := template.ParseFiles("templates/reset_link.html")
	if err != nil {
		fmt.Println(err)
		return err
	}

	data := map[string]interface{}{
		"ResetURL": resetURL,
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	return SendEmail(email, "重置密码", body.String())
}

// GenerateResetToken 生成重置密码的随机 token
func GenerateResetToken() (string, error) {
	bytes := make([]byte, 16) // 32 字节 = 64 位 hex
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
