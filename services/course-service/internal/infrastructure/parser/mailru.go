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
			Type string `json:"type"` // video, image, archive...
			Kind string `json:"kind"` // file или folder <--- ВОТ ЭТО НАМ НУЖНО
		} `json:"list"`
	} `json:"body"`
}

type LessonDTO struct {
	Title    string
	FileLink string
}

func ParseFolder(publicLink string) ([]LessonDTO, error) {
	// 1. Извлекаем weblink
	parts := strings.Split(publicLink, "/public/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid mail.ru link")
	}
	weblink := parts[1]

	// 2. Запрос к API
	apiURL := fmt.Sprintf("https://cloud.mail.ru/api/v2/folder?weblink=%s", weblink)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Обязательно User-Agent, иначе 403
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mail.ru api status: %d", resp.StatusCode)
	}

	// 3. Декодинг
	var mrResp MailRuResponse
	if err := json.NewDecoder(resp.Body).Decode(&mrResp); err != nil {
		return nil, err
	}

	var lessons []LessonDTO
	cleanBaseLink := strings.TrimRight(publicLink, "/")

	// 4. Фильтрация
	for _, item := range mrResp.Body.List {
		// ИСПРАВЛЕНИЕ: Проверяем Kind, а не Type
		// Kind = "file" (для видео, pdf, zip...)
		// Kind = "folder" (для папок)
		if item.Kind == "file" {
			fullLink := fmt.Sprintf("%s/%s", cleanBaseLink, item.Name)

			title := item.Name
			// Убираем расширение для красоты
			if idx := strings.LastIndex(title, "."); idx != -1 {
				title = title[:idx]
			}

			lessons = append(lessons, LessonDTO{
				Title:    title,
				FileLink: fullLink,
			})
		}
	}

	// Если в корне нет файлов (только папка), пробуем зайти внутрь этой папки?
	// Пока оставим так. Если files == 0, значит ссылка ведет на папку с папками.
	if len(lessons) == 0 {
		return nil, fmt.Errorf("файлы не найдены (проверьте, что они лежат в корне ссылки)")
	}

	return lessons, nil
}
