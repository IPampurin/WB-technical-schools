package robots

import (
	"net/url"
	"strings"
	"sync"
)

// robot - парсер robots.txt
type Robot struct {
	rules map[string][]string // правила: user-agent -> запрещенные пути
	mu    sync.RWMutex
}

// new создает новый экземпляр Robot
func New() *Robot {

	return &Robot{
		rules: make(map[string][]string),
	}
}

// parse парсит файл robots.txt
func (r *Robot) Parse(robotsContent string) {

	r.mu.Lock()
	defer r.mu.Unlock()

	lines := strings.Split(robotsContent, "\n")
	var currentAgent string

	for _, line := range lines {
		line = strings.TrimSpace(strings.Split(line, "#")[0]) // убираем комментарии
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "user-agent":
			// задаем текущего user-agent для следующих правил
			currentAgent = value
		case "disallow":
			// добавляем запрещенный путь для текущего user-agent
			if currentAgent != "" && value != "" {
				r.rules[currentAgent] = append(r.rules[currentAgent], value)
			}
		}
	}
}

// isAllowed проверяет, разрешен ли данный URL для указанного user-agent
func (r *Robot) IsAllowed(agent string, url *url.URL) bool {

	r.mu.RLock()
	defer r.mu.RUnlock()

	// проверяем все правила для агента и для "*"
	agents := []string{agent, "*"}

	for _, a := range agents {
		if disallows, ok := r.rules[a]; ok {
			path := url.Path
			for _, disallow := range disallows {
				// если путь начинается с запрещенного, доступ запрещен
				if strings.HasPrefix(path, disallow) {
					return false
				}
			}
		}
	}

	return true
}
