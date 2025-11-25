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

// Обновили структуру, добавили поле Weblink
type MailRuItem struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Kind    string `json:"kind"`
	Weblink string `json:"weblink"` // Важное поле для рекурсии и ссылок
}

type MailRuResponse struct {
	Body struct {
		List []MailRuItem `json:"list"`
	} `json:"body"`
}

type LessonDTO struct {
	Title    string
	FileLink string
}

// ParseFolder - входная точка
func ParseFolder(publicLink string) ([]LessonDTO, error) {
	log.Printf("====== [PARSER START RECURSIVE] ======")

	// 1. Извлекаем начальный weblink (например, 8CG5/yHMEs5Q88)
	parts := strings.Split(publicLink, "/public/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid mail.ru link")
	}
	rootWeblink := parts[1]

	// 2. Запускаем рекурсивный сбор
	lessons, err := fetchFolderRecursive(rootWeblink)
	if err != nil {
		return nil, err
	}

	log.Printf("Total lessons parsed: %d", len(lessons))
	log.Printf("====== [PARSER END] ======")

	if len(lessons) == 0 {
		return nil, fmt.Errorf("0 files found in folder structure")
	}

	return lessons, nil
}

// fetchFolderRecursive - рекурсивная функция
func fetchFolderRecursive(weblink string) ([]LessonDTO, error) {
	log.Printf("Scanning folder: %s", weblink)

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
		log.Printf("[PARSER ERROR] Network error for %s: %v", weblink, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Если ошибка доступа к подпапке, логируем, но не ломаем весь процесс (возвращаем пусто)
		log.Printf("[PARSER WARN] Status %d for %s", resp.StatusCode, weblink)
		return nil, nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var mrResp MailRuResponse
	if err := json.Unmarshal(bodyBytes, &mrResp); err != nil {
		return nil, err
	}

	var result []LessonDTO

	for _, item := range mrResp.Body.List {
		// Логика определения файла
		isTargetFile := false
		if item.Kind == "file" {
			isTargetFile = true
		}
		// На случай если kind пустой, смотрим type
		if item.Type == "video" || item.Type == "file" {
			isTargetFile = true
		}
		// Исключаем архивы и текстовые файлы, если нужно (опционально)
		if strings.HasSuffix(item.Name, ".zip") || strings.HasSuffix(item.Name, ".txt") {
			isTargetFile = false
		}

		if isTargetFile {
			// Формируем прямую публичную ссылку на файл
			// API возвращает weblink вида "part1/part2/file.mp4"
			// Нам нужно склеить с базовым доменом
			fullLink := fmt.Sprintf("https://cloud.mail.ru/public/%s", item.Weblink)

			title := item.Name
			// Убираем расширение из названия
			if idx := strings.LastIndex(title, "."); idx != -1 {
				title = title[:idx]
			}

			result = append(result, LessonDTO{
				Title:    title,
				FileLink: fullLink,
			})
			log.Printf("  -> Found file: %s", title)
		} else if item.Kind == "folder" || item.Type == "folder" {
			// === РЕКУРСИЯ ===
			// Если это папка, вызываем эту же функцию для нее
			subLessons, err := fetchFolderRecursive(item.Weblink)
			if err == nil {
				result = append(result, subLessons...)
			}
		}
	}

	return result, nil
}
