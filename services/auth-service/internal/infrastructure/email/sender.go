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
							background-color: #f5f5f5;
							margin: 0;
							padding: 0;
						}
						.container {
							max-width: 600px;
							margin: 50px auto;
							background-color: #ffffff;
							padding: 30px;
							border-radius: 10px;
							box-shadow: 0 4px 6px rgba(0,0,0,0.1);
						}
						h3 {
							color: #333333;
							text-align: center;
						}
						p {
							color: #555555;
							font-size: 16px;
							line-height: 1.5;
							text-align: center;
						}
						.button {
							display: block;
							width: 200px;
							margin: 30px auto;
							padding: 15px 0;
							background-color: #007BFF;
							color: #ffffff;
							text-decoration: none;
							text-align: center;
							border-radius: 5px;
							font-weight: bold;
							font-size: 16px;
						}
						.footer {
							text-align: center;
							font-size: 12px;
							color: #999999;
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
