package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"mywget/pkg/filesystem"
	"mywget/pkg/linkprocessor"
	"mywget/pkg/parser"
	"mywget/pkg/queue"
	"mywget/pkg/robots"
)

// Loader - структура загрузчика, управляющая процессом загрузки ресурса
type Loader struct {
	ctx       context.Context
	baseURL   *url.URL
	maxDepth  int
	visited   *sync.Map     // thread-safe мапа посещенных URL
	loadQueue *queue.Queue  // очередь для BFS обхода
	client    *http.Client  // HTTP клиент с таймаутами
	semaphore chan struct{} // семафор для ограничения параллельных загрузок
	robot     *robots.Robot //
	wg        *sync.WaitGroup
}

// Load запускает процесс загрузки сайта
func Load(ctx context.Context, startURL string, depth int) error {

	// 1. валидируем и парсим URL
	baseURL, err := url.Parse(startURL)
	if err != nil {
		return fmt.Errorf("неверный URL: %w", err)
	}

	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return fmt.Errorf("поддерживаются только HTTP/HTTPS протоколы")
	}

	if depth < 0 {
		return fmt.Errorf("глубина не может быть отрицательной")
	}

	// 2. создаем экземпляр загрузчика
	loader := &Loader{
		ctx:       ctx,
		baseURL:   baseURL,
		maxDepth:  depth,
		visited:   &sync.Map{},
		loadQueue: queue.New(),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		semaphore: make(chan struct{}, 10), // максимум 10 параллельных загрузок
		robot:     robots.New(),
		wg:        &sync.WaitGroup{},
	}

	// 3. загружаем robots.txt
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", baseURL.Scheme, baseURL.Hostname())
	if err := loader.loadRobotsTxt(robotsURL); err != nil {
		fmt.Printf("Предупреждение: не удалось загрузить robots.txt: %v\n", err)
	}

	// 4. запускаем загрузчик
	return loader.run()
}

// loadRobotsTxt загружает и парсит robots.txt
func (l *Loader) loadRobotsTxt(robotsURL string) error {

	req, err := http.NewRequestWithContext(l.ctx, "GET", robotsURL, nil)
	if err != nil {
		return err
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("robots.txt недоступен: статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024)) // 64KB максимум
	if err != nil {
		return err
	}

	l.robot.Parse(string(body))

	return nil
}

// run запускает основной цикл загрузки
func (l *Loader) run() error {

	fmt.Printf("Начинаем загрузку %s (глубина: %d)\n", l.baseURL.String(), l.maxDepth)

	// добавляем начальную страницу в очередь на загрузку
	l.loadQueue.Push(queue.Task{
		URL:   l.baseURL.String(),
		Depth: 0,
		Type:  "html",
	})

	// запускаем воркеры для параллельной загрузки
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		l.wg.Add(1)
		go l.worker(i + 1)
	}

	// ждем завершения всех воркеров
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	// ожидаем завершения или отмены контекста
	select {
	case <-done:
		fmt.Println("Загрузка завершена")
		return nil
	case <-l.ctx.Done():
		fmt.Println("Получен сигнал остановки...")
		// дожидаемся завершения текущих загрузок
		l.wg.Wait()
		return fmt.Errorf("загрузка прервана: %w", l.ctx.Err())
	}
}

// worker обрабатывает задачи из очереди
func (l *Loader) worker(id int) {

	defer l.wg.Done()

	for {
		select {
		case <-l.ctx.Done():
			// контекст отменен, выходим
			return
		default:
			// пытаемся взять задачу из очереди
			task, ok := l.loadQueue.Pop()
			if !ok {
				// очередь пуста, завершаем работу
				return
			}

			// обрабатываем задачу
			l.processTask(task, id)
		}
	}
}

// processTask обрабатывает одну задачу загрузки
func (l *Loader) processTask(task queue.Task, workerID int) {

	// проверяем, не посещали ли уже этот URL
	if _, visited := l.visited.Load(task.URL); visited {
		fmt.Printf("[Воркер %d] Пропускаем уже посещенный URL: %s\n", workerID, task.URL)
		return
	}

	// проверяем robots.txt
	parsedURL, err := url.Parse(task.URL)
	if err != nil {
		fmt.Printf("[Воркер %d] Ошибка парсинга URL: %v\n", workerID, err)
		return
	}

	if !l.robot.IsAllowed("mywget-bot", parsedURL) {
		fmt.Printf("[Воркер %d] Заблокировано robots.txt: %s\n", workerID, task.URL)
		return
	}

	// проверяем глубину
	if task.Depth > l.maxDepth {
		fmt.Printf("[Воркер %d] Пропускаем, превышена глубина: %s (глубина %d)\n",
			workerID, task.URL, task.Depth)
		return
	}

	// проверяем, что URL принадлежит тому же домену
	if !l.isSameDomain(task.URL) {
		fmt.Printf("[Воркер %d] Пропускаем внешний домен: %s\n", workerID, task.URL)
		return
	}

	// занимаем слот семафора для ограничения параллелизма
	l.semaphore <- struct{}{}
	defer func() { <-l.semaphore }()

	// помечаем URL как посещенный
	l.visited.Store(task.URL, true)

	fmt.Printf("[Воркер %d] Начинаю загрузку: %s (тип: %s, глубина: %d)\n",
		workerID, task.URL, task.Type, task.Depth)

	// выполняем запрос
	req, err := http.NewRequestWithContext(l.ctx, "GET", task.URL, nil)
	if err != nil {
		fmt.Printf("[Воркер %d] Ошибка создания запроса: %v\n", workerID, err)
		return
	}

	resp, err := l.client.Do(req)
	if err != nil {
		fmt.Printf("[Воркер %d] Ошибка загрузки: %v\n", workerID, err)
		return
	}
	defer resp.Body.Close()

	// проверяем статус
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[Воркер %d] Пропускаем, статус: %s\n", workerID, resp.Status)
		return
	}

	// ограничиваем размер загружаемого файла
	var maxSize int64
	switch task.Type {
	case "html", "css", "js":
		maxSize = 10 * 1024 * 1024 // 10 МБ
	default:
		maxSize = 100 * 1024 * 1024 // 100 МБ
	}

	// читаем тело с ограничением
	limitedBody := io.LimitReader(resp.Body, maxSize)
	bodyBytes, err := io.ReadAll(limitedBody)
	if err != nil {
		fmt.Printf("[Воркер %d] Ошибка чтения тела: %v\n", workerID, err)
		return
	}

	// также проверяем, не превышен ли лимит
	if resp.ContentLength > maxSize {
		fmt.Printf("[Воркер %d] Файл слишком большой: %d байт\n", workerID, resp.ContentLength)
		return
	}

	// определяем Content-Type
	contentType := resp.Header.Get("Content-Type")
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mimeType = contentType
	}

	// сохраняем файл
	bodyReader := bytes.NewReader(bodyBytes)
	localPath, err := filesystem.SaveFile(task.URL, l.baseURL, bodyReader, mimeType)
	if err != nil {
		fmt.Printf("[Воркер %d] Ошибка сохранения: %v\n", workerID, err)
		return
	}

	fmt.Printf("[Воркер %d] Сохранен: %s -> %s\n", workerID, task.URL, localPath)

	// если это HTML и глубина позволяет, парсим и добавляем ссылки
	if mimeType == "text/html" && task.Depth < l.maxDepth {
		// заменяем ссылки в HTML на локальные
		pageURL, _ := url.Parse(task.URL)
		newBodyBytes, err := linkprocessor.ReplaceLinks(bodyBytes, pageURL, l.baseURL)
		if err != nil {
			fmt.Printf("[Воркер %d] Ошибка замены ссылок: %v\n", workerID, err)
			return
		}

		// перезаписываем файл с замененными ссылками
		newBodyReader := bytes.NewReader(newBodyBytes)
		localPath, err = filesystem.SaveFile(task.URL, l.baseURL, newBodyReader, mimeType)
		if err != nil {
			fmt.Printf("[Воркер %d] Ошибка перезаписи файла: %v\n", workerID, err)
			return
		}

		// парсим ссылки из исходного HTML (до замены)
		pageLinks, resourceLinks, err := parser.ExtractLinks(pageURL, bytes.NewReader(bodyBytes))
		if err != nil {
			fmt.Printf("[Воркер %d] Ошибка парсинга HTML: %v\n", workerID, err)
			return
		}

		// добавляем ресурсы (глубина не увеличивается)
		for _, resURL := range resourceLinks {
			if !l.isSameDomain(resURL) {
				continue
			}
			if _, visited := l.visited.Load(resURL); visited {
				continue
			}
			resType := l.getResourceType(resURL)
			l.loadQueue.Push(queue.Task{
				URL:   resURL,
				Depth: task.Depth,
				Type:  resType,
			})
		}

		// добавляем страницы (глубина +1)
		for _, pageURL := range pageLinks {
			if !l.isSameDomain(pageURL) {
				continue
			}
			if _, visited := l.visited.Load(pageURL); visited {
				continue
			}
			l.loadQueue.Push(queue.Task{
				URL:   pageURL,
				Depth: task.Depth + 1,
				Type:  "html",
			})
		}
	}
}

// getResourceType определяет тип ресурса по URL
func (l *Loader) getResourceType(urlStr string) string {

	u, err := url.Parse(urlStr)
	if err != nil {
		return "unknown"
	}

	path := strings.ToLower(u.Path)

	if strings.HasSuffix(path, ".css") {
		return "css"
	} else if strings.HasSuffix(path, ".js") {
		return "js"
	} else if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif") || strings.HasSuffix(path, ".svg") {
		return "image"
	} else if strings.HasSuffix(path, ".woff") || strings.HasSuffix(path, ".woff2") || strings.HasSuffix(path, ".ttf") || strings.HasSuffix(path, ".eot") {
		return "font"
	} else {
		return "unknown"
	}
}

// isSameDomain проверяет, принадлежит ли URL тому же домену
func (l *Loader) isSameDomain(urlStr string) bool {

	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// сравниваем домены (без учета протокола и порта)
	return u.Hostname() == l.baseURL.Hostname()
}
