package email

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"
)

var (
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	fromEmail    string
	fromName     string
)

// InitEmail 初始化邮件配置
func InitEmail() {
	smtpHost = os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.qq.com"
	}
	smtpPort = os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}
	smtpUser = os.Getenv("SMTP_USER")
	smtpPassword = os.Getenv("SMTP_PASSWORD")
	fromEmail = os.Getenv("SMTP_FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = smtpUser
	}
	fromName = os.Getenv("SMTP_FROM_NAME")
	if fromName == "" {
		fromName = "小野 AI"
	}

	if smtpUser != "" && smtpPassword != "" {
		log.Printf("邮件服务已配置: %s:%s", smtpHost, smtpPort)
	} else {
		log.Printf("警告: 邮件服务未配置 (SMTP_USER/SMTP_PASSWORD 未设置)")
	}
}

// IsConfigured 检查邮件服务是否配置
func IsConfigured() bool {
	return smtpUser != "" && smtpPassword != ""
}

// SendVerificationCode 发送验证码邮件
func SendVerificationCode(to, code, purpose string) error {
	if !IsConfigured() {
		log.Printf("邮件服务未配置，验证码 [%s] 发送到 [%s] (类型: %s)", code, to, purpose)
		return nil // 在开发环境下不报错
	}

	var subject, body string

	switch purpose {
	case "register":
		subject = "小野 AI - 注册验证码"
		body = fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center;">
					<h1 style="color: white; margin: 0;">🎨 小野 AI</h1>
				</div>
				<div style="padding: 30px; background: #f9f9f9;">
					<h2 style="color: #333;">欢迎注册 小野 AI</h2>
					<p style="color: #666;">您的验证码是：</p>
					<div style="background: white; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0;">
						<span style="font-size: 32px; font-weight: bold; color: #667eea; letter-spacing: 8px;">%s</span>
					</div>
					<p style="color: #999; font-size: 14px;">验证码有效期为10分钟，请尽快使用。</p>
					<p style="color: #999; font-size: 14px;">如果这不是您的操作，请忽略此邮件。</p>
				</div>
				<div style="background: #333; padding: 20px; text-align: center;">
					<p style="color: #999; margin: 0; font-size: 12px;">© %d 小野 AI. All rights reserved.</p>
				</div>
			</body>
			</html>
		`, code, time.Now().Year())
	case "login":
		subject = "小野 AI - 登录验证码"
		body = fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center;">
					<h1 style="color: white; margin: 0;">🎨 小野 AI</h1>
				</div>
				<div style="padding: 30px; background: #f9f9f9;">
					<h2 style="color: #333;">登录验证</h2>
					<p style="color: #666;">您的登录验证码是：</p>
					<div style="background: white; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0;">
						<span style="font-size: 32px; font-weight: bold; color: #667eea; letter-spacing: 8px;">%s</span>
					</div>
					<p style="color: #999; font-size: 14px;">验证码有效期为10分钟，请尽快使用。</p>
					<p style="color: #999; font-size: 14px;">如果这不是您的操作，请忽略此邮件。</p>
				</div>
				<div style="background: #333; padding: 20px; text-align: center;">
					<p style="color: #999; margin: 0; font-size: 12px;">© %d 小野 AI. All rights reserved.</p>
				</div>
			</body>
			</html>
		`, code, time.Now().Year())
	case "reset":
		subject = "小野 AI - 重置密码验证码"
		body = fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center;">
					<h1 style="color: white; margin: 0;">🎨 小野 AI</h1>
				</div>
				<div style="padding: 30px; background: #f9f9f9;">
					<h2 style="color: #333;">重置密码</h2>
					<p style="color: #666;">您正在重置密码，验证码是：</p>
					<div style="background: white; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0;">
						<span style="font-size: 32px; font-weight: bold; color: #667eea; letter-spacing: 8px;">%s</span>
					</div>
					<p style="color: #999; font-size: 14px;">验证码有效期为10分钟，请尽快使用。</p>
					<p style="color: #999; font-size: 14px;">如果这不是您的操作，请立即检查账号安全。</p>
				</div>
				<div style="background: #333; padding: 20px; text-align: center;">
					<p style="color: #999; margin: 0; font-size: 12px;">© %d 小野 AI. All rights reserved.</p>
				</div>
			</body>
			</html>
		`, code, time.Now().Year())
	default:
		subject = "小野 AI - 验证码"
		body = fmt.Sprintf(`您的验证码是: %s，有效期10分钟。`, code)
	}

	return sendHTML(to, subject, body)
}

// sendHTML 发送HTML格式邮件
func sendHTML(to, subject, body string) error {
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	err := smtp.SendMail(addr, auth, fromEmail, []string{to}, []byte(message))
	if err != nil {
		// QQ邮箱的SMTP服务器有时会在邮件发送成功后返回不完整的响应
		// 导致 "short response" 错误，但邮件实际上已经发送成功
		// 参考: https://github.com/golang/go/issues/24845
		errStr := err.Error()
		if strings.HasPrefix(errStr, "short response") {
			log.Printf("邮件已发送成功 (忽略 short response 错误): %s <- %s", to, subject)
			return nil
		}
		log.Printf("发送邮件失败 [%s]: %v", to, err)
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	log.Printf("邮件发送成功: %s <- %s", to, subject)
	return nil
}
