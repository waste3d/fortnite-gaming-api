package email

import (
	"fmt"
	"net/smtp"
)

type EmailSender struct {
	host     string
	port     string
	email    string
	password string
	frontend string
}

func NewEmailSender(host, port, email, password, frontend string) *EmailSender {
	return &EmailSender{
		host:     host,
		port:     port,
		email:    email,
		password: password,
		frontend: frontend,
	}
}

func (s *EmailSender) SendResetEmail(toEmail, token string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontend, token)

	subject := "Subject: Password Reset Request\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
	<html>
	<body>
		<h3>Восстановление пароля</h3>
		<p>Вы запросили сброс пароля. Нажмите на ссылку ниже:</p>
		<p><a href="%s">Сбросить пароль</a></p>
		<p>Ссылка действительна 15 минут.</p>
	</body>
	</html>
`, resetLink)

	msg := []byte(subject + mime + body)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.email, s.password, s.host)

	return smtp.SendMail(addr, auth, s.email, []string{toEmail}, msg)
}
