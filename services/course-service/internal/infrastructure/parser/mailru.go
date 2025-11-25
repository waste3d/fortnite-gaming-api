package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Структура ответа (попытка угадать поля)
type MailRuResponse struct {
	Body struct {
		List []struct {
			Name string `json:"name"`
			Type string `json:"type"` // video, image, file, folder
			Kind string `json:"kind"` // file, folder
		} `json:"list"`
	} `json:"body"`
}

type LessonDTO struct {
	Title    string
	FileLink string
}

func ParseFolder(publicLink string) ([]LessonDTO, error) {
	log.Printf("====== [PARSER START] ======")
	log.Printf("Processing link: %s", publicLink)

	// 1. Извлекаем weblink
	parts := strings.Split(publicLink, "/public/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid mail.ru link")
	}
	weblink := parts[1]
	log.Printf("Extracted weblink ID: %s", weblink)

	// 2. Запрос к API
	apiURL := fmt.Sprintf("https://cloud.mail.ru/api/v2/folder?weblink=%s", weblink)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[PARSER ERROR] Network error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("API Status Code: %d", resp.StatusCode)

	// 3. Читаем сырой ответ (чтобы видеть в логах)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// ЛОГГИРУЕМ СЫРОЙ JSON!
	log.Printf("--- RAW JSON RESPONSE START ---")
	log.Printf("%s", string(bodyBytes))
	log.Printf("--- RAW JSON RESPONSE END ---")

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mail.ru api returned status: %d", resp.StatusCode)
	}

	// 4. Декодируем
	var mrResp MailRuResponse
	if err := json.Unmarshal(bodyBytes, &mrResp); err != nil {
		log.Printf("[PARSER ERROR] JSON Decode failed: %v", err)
		return nil, err
	}

	var lessons []LessonDTO
	cleanBaseLink := strings.TrimRight(publicLink, "/")

	log.Printf("Found items in list: %d", len(mrResp.Body.List))

	// 5. Фильтрация и дебаг каждого элемента
	for i, item := range mrResp.Body.List {
		log.Printf("Item [%d]: Name='%s', Type='%s', Kind='%s'", i, item.Name, item.Type, item.Kind)

		// Пробуем разные варианты проверки
		isTargetFile := false

		// Вариант 1: Kind == file
		if item.Kind == "file" {
			isTargetFile = true
		}
		// Вариант 2: Type == video (если kind пустой или другой)
		if item.Type == "video" || item.Type == "file" {
			isTargetFile = true
		}

		if isTargetFile {
			fullLink := fmt.Sprintf("%s/%s", cleanBaseLink, item.Name)
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

	log.Printf("Total lessons parsed: %d", len(lessons))
	log.Printf("====== [PARSER END] ======")

	if len(lessons) == 0 {
		return nil, fmt.Errorf("0 files found in folder")
	}

	return lessons, nil
}
