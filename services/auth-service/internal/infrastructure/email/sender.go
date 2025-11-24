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
				<html>
				<head>
					<style>
						body {
							font-family: Arial, sans-serif;
							background-color: #0d1b2a;
							margin: 0;
							padding: 0;
							color: #ffffff;
						}
						.container {
							max-width: 600px;
							margin: 50px auto;
							background-color: #1b263b;
							padding: 30px;
							border-radius: 12px;
							box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
							text-align: center;
						}
						h3 {
							color: #ffffff;
							margin-bottom: 20px;
						}
						p {
							color: #d1d1d1;
							font-size: 16px;
							line-height: 1.5;
						}
						.button {
							display: inline-block;
							margin: 30px 0;
							padding: 15px 30px;
							background-color: #1b263b;
							border: 2px solid #ff4d4d;
							color: #ff4d4d;
							text-decoration: none;
							font-weight: bold;
							font-size: 16px;
							border-radius: 6px;
							box-shadow: 0 0 15px #ff4d4d;
							transition: all 0.3s ease;
						}
						.button:hover {
							background-color: #ff4d4d;
							color: #1b263b;
							box-shadow: 0 0 25px #ff1a1a;
						}
						.footer {
							font-size: 12px;
							color: #888888;
							margin-top: 20px;
						}
					</style>
				</head>
				<body>
					<div class="container">
						<h3>Сброс пароля</h3>
						<p>Вы запросили сброс пароля. Нажмите на кнопку ниже, чтобы установить новый пароль.</p>
						<a href="%s" class="button">Сбросить пароль</a>
						<p class="footer">Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.</p>
					</div>
				</body>
				</html>
				`, resetLink),
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
