package filesystem

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SaveFile сохраняет тело ответа в файловую систему
func SaveFile(fileURL string, baseURL *url.URL, body io.Reader, contentType string) (string, error) {

	u, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("parse file URL: %w", err)
	}

	localPath, err := GetLocalPath(u, baseURL, contentType)
	if err != nil {
		return "", fmt.Errorf("get local path: %w", err)
	}

	// создаем директорию
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory %s: %w", dir, err)
	}

	// создаем файл
	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create file %s: %w", localPath, err)
	}
	defer file.Close()

	// копируем тело
	if _, err := io.Copy(file, body); err != nil {
		return "", fmt.Errorf("write file %s: %w", localPath, err)
	}

	return localPath, nil
}

// GetLocalPath преобразует URL в локальный путь (относительно текущей директории)
func GetLocalPath(fileURL, baseURL *url.URL, contentType string) (string, error) {

	if fileURL.Hostname() != baseURL.Hostname() {
		return "", fmt.Errorf("host mismatch: %s != %s", fileURL.Hostname(), baseURL.Hostname())
	}

	path := fileURL.Path
	// если путь пустой или заканчивается на / — добавляем index.html
	if path == "" || strings.HasSuffix(path, "/") {
		path = path + "index.html"
	} else {
		// если это HTML-страница без расширения — добавляем .html
		if strings.Contains(contentType, "text/html") && !strings.Contains(filepath.Base(path), ".") {
			path = path + ".html"
		}
	}

	// собираем путь: host + path
	localPath := filepath.Join(fileURL.Hostname(), path)
	localPath = filepath.Clean(localPath)

	return localPath, nil
}
