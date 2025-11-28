package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MailRuItem описывает один элемент (файл или папку) в ответе API
type MailRuItem struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // video, image, file, folder
	Kind    string `json:"kind"`    // file, folder
	Weblink string `json:"weblink"` // Путь внутри облака
}

// MailRuResponse структура ответа от API
type MailRuResponse struct {
	Body struct {
		List []MailRuItem `json:"list"`
	} `json:"body"`
}

// LessonDTO результат парсинга для сохранения в БД
type LessonDTO struct {
	Title    string
	FileLink string
}

// ParseFolder - основная точка входа
func ParseFolder(publicLink string) ([]LessonDTO, error) {
	log.Printf("====== [PARSER START RECURSIVE v2] ======")
	log.Printf("Processing root link: %s", publicLink)

	// 1. Извлекаем корневой weblink (идентификатор папки)
	// Ссылка вида https://cloud.mail.ru/public/8CG5/yHMEs5Q88
	parts := strings.Split(publicLink, "/public/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid mail.ru link format")
	}
	rootWeblink := parts[1]

	// 2. Запускаем рекурсивный обход
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

// fetchFolderRecursive - рекурсивно обходит папки
func fetchFolderRecursive(weblink string) ([]LessonDTO, error) {
	log.Printf("Scanning folder: %s", weblink)

	// 1. Подготовка URL с правильным кодированием параметров (для пробелов и кириллицы)
	baseURL := "https://cloud.mail.ru/api/v2/folder"
	params := url.Values{}
	params.Add("weblink", weblink)

	// Собираем итоговый URL: .../folder?weblink=encoded_path
	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// 2. Создание запроса
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Обязательно притворяемся браузером
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[PARSER ERROR] Network error for %s: %v", weblink, err)
		return nil, err
	}
	defer resp.Body.Close()

	// 3. Обработка ошибок HTTP
	if resp.StatusCode != 200 {
		log.Printf("[PARSER WARN] Status %d for %s", resp.StatusCode, weblink)
		// Если не удалось прочитать папку, возвращаем пустой список, но не ломаем весь процесс
		return nil, nil
	}

	// 4. Декодирование JSON
	bodyBytes, _ := io.ReadAll(resp.Body)
	var mrResp MailRuResponse
	if err := json.Unmarshal(bodyBytes, &mrResp); err != nil {
		log.Printf("[PARSER ERROR] JSON Decode failed: %v", err)
		return nil, err
	}

	var result []LessonDTO

	// 5. Перебор элементов
	for _, item := range mrResp.Body.List {
		// Определение: Файл это или Папка?
		isTargetFile := false
		if item.Kind == "file" {
			isTargetFile = true
		}
		// Подстраховка по типу
		if item.Type == "video" || item.Type == "file" {
			isTargetFile = true
		}

		// Игнорируем архивные файлы, текстовые и pdf (если нужны только видео)
		lowerName := strings.ToLower(item.Name)
		if strings.HasSuffix(lowerName, ".url") || strings.HasSuffix(lowerName, ".docx") {
			isTargetFile = false
		}

		if isTargetFile {
			// === ЭТО ФАЙЛ ===
			// Формируем прямую ссылку. Обычно достаточно склеить base url и weblink файла.
			// Пример weblink файла: "8CG5/yHMEs5Q88/Модуль 1/video.mp4"
			fullLink := fmt.Sprintf("https://cloud.mail.ru/public/%s", item.Weblink)

			// Очищаем название от расширения
			title := item.Name
			if idx := strings.LastIndex(title, "."); idx != -1 {
				title = title[:idx]
			}

			result = append(result, LessonDTO{
				Title:    title,
				FileLink: fullLink,
			})
			log.Printf("  -> Found file: %s", title)

		} else if item.Kind == "folder" || item.Type == "folder" {
			// === ЭТО ПАПКА (РЕКУРСИЯ) ===
			// Вызываем эту же функцию для вложенной папки
			subLessons, err := fetchFolderRecursive(item.Weblink)
			if err == nil {
				result = append(result, subLessons...)
			}
		}
	}

	return result, nil
}
