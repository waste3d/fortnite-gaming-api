package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type EmailSender struct {
	apiKey      string
	senderEmail string
	senderName  string
	frontend    string
}

func NewEmailSender(apiKey, senderEmail, frontend string) *EmailSender {
	return &EmailSender{
		apiKey:      apiKey,
		senderEmail: senderEmail,
		senderName:  "BazaKursov Support",
		frontend:    frontend,
	}
}

// SendGrid request format
type sgEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}
type sgContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
type sgRequest struct {
	Personalizations []struct {
		To []sgEmail `json:"to"`
	} `json:"personalizations"`
	From    sgEmail     `json:"from"`
	Subject string      `json:"subject"`
	Content []sgContent `json:"content"`
}

func (s *EmailSender) SendResetEmail(toEmail string, token string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontend, token)

	body := sgRequest{
		Personalizations: []struct {
			To []sgEmail `json:"to"`
		}{
			{To: []sgEmail{{Email: toEmail}}},
		},
		From: sgEmail{
			Email: s.senderEmail,
			Name:  s.senderName,
		},
		Subject: "Восстановление пароля",
		Content: []sgContent{
			{
				Type: "text/html",
				Value: fmt.Sprintf(`
				<!DOCTYPE html>
				<html>
				<head>
					<meta charset="utf-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<title>Сброс пароля</title>
					<style>
						body {
							font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif;
							background-color: #0f0f11; /* Твой основной темный фон */
							margin: 0;
							padding: 0;
							color: #ffffff;
						}
						.wrapper {
							width: 100%%;
							table-layout: fixed;
							background-color: #0f0f11;
							padding-bottom: 40px;
						}
						.container {
							max-width: 480px;
							margin: 40px auto;
							background-color: #18181b; /* Цвет карточек */
							padding: 40px;
							border-radius: 16px; /* Скругление как в интерфейсе */
							border: 1px solid #27272a; /* Тонкая рамка */
							box-shadow: 0 4px 30px rgba(0, 0, 0, 0.5);
							text-align: center;
						}
						.logo {
							margin-bottom: 30px;
							font-size: 24px;
							font-weight: 800;
							letter-spacing: -0.5px;
							text-decoration: none;
						}
						.logo-white { color: #ffffff; }
						.logo-teal { color: #2dd4bf; } /* Твой accent-teal */
						
						h3 {
							color: #ffffff;
							margin-top: 0;
							margin-bottom: 16px;
							font-size: 20px;
						}
						p {
							color: #a1a1aa; /* Серый текст (text-gray-400) */
							font-size: 15px;
							line-height: 1.6;
							margin-bottom: 30px;
						}
						.button-container {
							margin: 30px 0;
						}
						.button {
							display: inline-block;
							padding: 14px 32px;
							background-color: #2dd4bf; /* Teal кнопка */
							color: #ffffff;
							text-decoration: none;
							font-weight: 600;
							font-size: 15px;
							border-radius: 10px;
							transition: background-color 0.3s ease;
						}
						/* Для поддержки hover в некоторых клиентах */
						.button:hover {
							background-color: #14b8a6;
						}
						.footer {
							font-size: 12px;
							color: #52525b; /* Более темный серый для футера */
							margin-top: 40px;
							border-top: 1px solid #27272a;
							padding-top: 20px;
						}
						.link {
							color: #2dd4bf;
							text-decoration: none;
						}
					</style>
				</head>
				<body>
					<div class="wrapper">
						<div class="container">
							<!-- Логотип текстом, так надежнее для писем -->
							<div class="logo">
								<span class="logo-white">BAZA</span><span class="logo-teal">KURSOV</span>
							</div>
		
							<h3>Восстановление доступа</h3>
							
							<p>Мы получили запрос на сброс пароля для вашего аккаунта. <br>Если это были вы, нажмите на кнопку ниже:</p>
							
							<div class="button-container">
								<!-- Ссылка -->
								<a href="%s" class="button" target="_blank">Установить новый пароль</a>
							</div>
							
							<p style="margin-bottom: 0;">Если кнопка не работает, скопируйте ссылку в браузер:</p>
							<p style="font-size: 12px; word-break: break-all; margin-top: 10px;">
								<a href="%s" class="link">%s</a>
							</p>
		
							<div class="footer">
								Если вы не запрашивали смену пароля, просто проигнорируйте это письмо. Ваш аккаунт в безопасности.
							</div>
						</div>
					</div>
				</body>
				</html>
				`, resetLink, resetLink, resetLink),
			},
		},
	}

	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest(
		"POST",
		"https://api.sendgrid.com/v3/mail/send",
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// SendGrid возвращает 202 даже при успехе
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendgrid error: status=%d body=%s", resp.StatusCode, body)
	}

	return nil
}
