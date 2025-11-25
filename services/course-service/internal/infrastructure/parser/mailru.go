package parser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type MailRuResponse struct {
	Body struct {
		List []struct {
			Name string `json:"name"`
			Type string `json:"type"` // "file" или "folder"
		} `json:"list"`
	} `json:"body"`
}

type LessonDTO struct {
	Title    string
	FileLink string
}

// ParseFolder принимает ссылку вида https://cloud.mail.ru/public/ABC/xyz
func ParseFolder(publicLink string) ([]LessonDTO, error) {
	// 1. Достаем "weblink" (код папки)
	parts := strings.Split(publicLink, "/public/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid mail.ru link")
	}
	weblink := parts[1]

	// 2. Идем в скрытое API
	apiURL := fmt.Sprintf("https://cloud.mail.ru/api/v2/folder?weblink=%s", weblink)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mail.ru api error: %d", resp.StatusCode)
	}

	var mrResp MailRuResponse
	if err := json.NewDecoder(resp.Body).Decode(&mrResp); err != nil {
		return nil, err
	}

	// 3. Формируем список
	var lessons []LessonDTO

	// Очищаем ссылку от возможных слешей в конце, чтобы красиво склеить
	cleanBaseLink := strings.TrimRight(publicLink, "/")

	for _, item := range mrResp.Body.List {
		if item.Type == "file" {
			// Ссылка на файл в облаке: Базовая_ссылка / Имя_файла
			fullLink := fmt.Sprintf("%s/%s", cleanBaseLink, item.Name)

			// Убираем расширение из названия для красоты (опционально)
			title := item.Name
			if idx := strings.LastIndex(title, "."); idx != -1 {
				title = title[:idx]
			}

			lessons = append(lessons, LessonDTO{
				Title:    title,
				FileLink: fullLink,
			})
		}
	}

	return lessons, nil
}
