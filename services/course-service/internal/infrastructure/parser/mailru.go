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

// ParseFolder получает список файлов из публичной ссылки
func ParseFolder(publicLink string) ([]LessonDTO, error) {
	// 1. Извлекаем weblink
	// Ссылка: https://cloud.mail.ru/public/82yg/pTapHZM29
	parts := strings.Split(publicLink, "/public/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("некорректная ссылка mail.ru")
	}
	weblink := parts[1]

	// 2. Формируем запрос к API
	apiURL := fmt.Sprintf("https://cloud.mail.ru/api/v2/folder?weblink=%s", weblink)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// ВАЖНО: Добавляем заголовки, чтобы нас не блокировали
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mail.ru api вернул статус: %d", resp.StatusCode)
	}

	// 3. Декодируем
	var mrResp MailRuResponse
	if err := json.NewDecoder(resp.Body).Decode(&mrResp); err != nil {
		return nil, err
	}

	var lessons []LessonDTO
	cleanBaseLink := strings.TrimRight(publicLink, "/")

	// 4. Собираем файлы
	for _, item := range mrResp.Body.List {
		if item.Type == "file" {
			// Формируем прямую ссылку на плеер/файл
			fullLink := fmt.Sprintf("%s/%s", cleanBaseLink, item.Name)

			// Убираем расширение из названия (.mp4, .pdf) для красоты
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

	// Если список пуст, возможно, это не корневая папка или структура другая
	if len(lessons) == 0 {
		return nil, fmt.Errorf("файлы не найдены (возможно, они во вложенных папках)")
	}

	return lessons, nil
}
