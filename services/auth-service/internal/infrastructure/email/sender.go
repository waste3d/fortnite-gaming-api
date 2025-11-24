package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type EmailSender struct {
	apiKey      string // Ваш API Key от Brevo
	senderEmail string // Почта отправителя (нужно подтвердить в Brevo)
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

// Структуры для JSON запроса Brevo
type brevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
type brevoTo struct {
	Email string `json:"email"`
}
type brevoRequest struct {
	Sender      brevoSender `json:"sender"`
	To          []brevoTo   `json:"to"`
	Subject     string      `json:"subject"`
	HtmlContent string      `json:"htmlContent"`
}

func (s *EmailSender) SendResetEmail(toEmail string, token string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontend, token)

	reqBody := brevoRequest{
		Sender:  brevoSender{Name: s.senderName, Email: s.senderEmail},
		To:      []brevoTo{{Email: toEmail}},
		Subject: "Восстановление пароля",
		HtmlContent: fmt.Sprintf(`
			<html><body>
				<h3>Сброс пароля</h3>
				<p>Нажмите ссылку ниже:</p>
				<a href="%s">Сбросить пароль</a>
			</body></html>`, resetLink),
	}

	bodyBytes, _ := json.Marshal(reqBody)

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("api-key", s.apiKey) // Авторизация через API Key
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to send email, status: %d", resp.StatusCode)
	}

	return nil
}
