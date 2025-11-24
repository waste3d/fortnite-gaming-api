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
				Type:  "text/html",
				Value: fmt.Sprintf(`<h3>Сброс пароля</h3><a href="%s">Сбросить пароль</a>`, resetLink),
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
