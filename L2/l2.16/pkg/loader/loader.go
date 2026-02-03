package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mywget/pkg/filesystem"
	"mywget/pkg/linkprocessor"
	"mywget/pkg/parser"
	"mywget/pkg/robots"
)

// task - задача для загрузки
type Task struct {
	URL   string
	Depth int
	Type  string // "html", "css", "js", "image", "font"
}

// loader - структура загрузчика, управляющая процессом загрузки ресурса
type Loader struct {
	ctx        context.Context
	baseURL    *url.URL
	maxDepth   int
	visited    *sync.Map       // мапа посещенных URL
	taskChan   chan Task       // канал задач для воркеров
	client     *http.Client    // HTTP клиент с таймаутами
	numWorkers int             // количество обработчиков задач
	robot      *robots.Robot   // парсер robots.txt
	wg         *sync.WaitGroup // для воркеров
	taskWg     *sync.WaitGroup // для отслеживания выполнения задач
}

// load запускает процесс загрузки сайта
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
		ctx:      ctx,
		baseURL:  baseURL,
		maxDepth: depth,
		visited:  &sync.Map{},
		taskChan: make(chan Task, 1000),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		numWorkers: 10,
		robot:      robots.New(),
		wg:         &sync.WaitGroup{},
		taskWg:     &sync.WaitGroup{},
	}

	// 3. загружаем robots.txt
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", baseURL.Scheme, baseURL.Hostname())
	if err := loader.loadRobotsTxt(robotsURL); err != nil {
		fmt.Printf("предупреждение: не удалось загрузить robots.txt: %v\n", err)
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

	fmt.Printf("начинаем загрузку %s (глубина: %d)\n", l.baseURL.String(), l.maxDepth)

	// запускаем воркеры для параллельной загрузки
	for i := 0; i < l.numWorkers; i++ {
		l.wg.Add(1)
		go l.worker(i + 1)
	}

	// ждём завершения всех задач и закрываем канал
	go func() {
		l.taskWg.Wait()
		close(l.taskChan)
	}()

	// добавляем начальную страницу в канал задач
	l.taskWg.Add(1)
	select {
	case l.taskChan <- Task{
		URL:   l.baseURL.String(),
		Depth: 0,
		Type:  "html"}:
	case <-l.ctx.Done():
		l.taskWg.Done()
		close(l.taskChan)
		l.wg.Wait()
		return fmt.Errorf("загрузка прервана: %w", l.ctx.Err())
	}

	// ждем завершения всех воркеров
	l.wg.Wait()

	// проверяем, не был ли отменен контекст
	if l.ctx.Err() != nil {
		return fmt.Errorf("загрузка прервана: %w", l.ctx.Err())
	}

	fmt.Println("загрузка завершена")

	return nil
}

// worker обрабатывает задачи из канала
func (l *Loader) worker(id int) {

	defer l.wg.Done()

	for {
		select {
		case <-l.ctx.Done():
			return
		case task, ok := <-l.taskChan:
			if !ok {
				return
			}
			l.processTask(task, id)
		}
	}
}

// processTask обрабатывает одну задачу загрузки
func (l *Loader) processTask(task Task, workerID int) {

	defer l.taskWg.Done() // уменьшаем счетчик задач при завершении

	// проверяем, не посещали ли уже этот URL
	if _, visited := l.visited.Load(task.URL); visited {
		fmt.Printf("[воркер %d] пропускаем уже посещенный URL: %s\n", workerID, task.URL)
		return
	}

	// проверяем глубину
	if task.Depth > l.maxDepth {
		fmt.Printf("[воркер %d] пропускаем, превышена глубина: %s (глубина %d)\n",
			workerID, task.URL, task.Depth)
		return
	}

	// проверяем, что URL принадлежит тому же домену
	if !l.isSameDomain(task.URL) {
		fmt.Printf("[воркер %d] пропускаем внешний домен: %s\n", workerID, task.URL)
		return
	}

	// проверяем robots.txt
	parsedURL, err := url.Parse(task.URL)
	if err != nil {
		fmt.Printf("[воркер %d] ошибка парсинга URL: %v\n", workerID, err)
		return
	}

	if !l.robot.IsAllowed("mywget-bot", parsedURL) {
		fmt.Printf("[воркер %d] заблокировано robots.txt: %s\n", workerID, task.URL)
		return
	}

	// помечаем URL как посещенный
	l.visited.Store(task.URL, true)

	fmt.Printf("[воркер %d] начинаю загрузку: %s (тип: %s, глубина: %d)\n",
		workerID, task.URL, task.Type, task.Depth)

	// выполняем запрос
	req, err := http.NewRequestWithContext(l.ctx, "GET", task.URL, nil)
	if err != nil {
		fmt.Printf("[воркер %d] ошибка создания запроса: %v\n", workerID, err)
		return
	}

	resp, err := l.client.Do(req)
	if err != nil {
		fmt.Printf("[воркер %d] ошибка загрузки: %v\n", workerID, err)
		return
	}
	defer resp.Body.Close()

	// проверяем статус
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[воркер %d] пропускаем, статус: %s\n", workerID, resp.Status)
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
		fmt.Printf("[воркер %d] ошибка чтения тела: %v\n", workerID, err)
		return
	}

	// также проверяем, не превышен ли лимит
	if resp.ContentLength > maxSize {
		fmt.Printf("[воркер %d] файл слишком большой: %d байт\n", workerID, resp.ContentLength)
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
		fmt.Printf("[воркер %d] ошибка сохранения: %v\n", workerID, err)
		return
	}

	fmt.Printf("[воркер %d] сохранен: %s -> %s\n", workerID, task.URL, localPath)

	// обрабатываем разные типы файлов
	switch {
	case mimeType == "text/html":
		l.processHTMLFile(task, bodyBytes, mimeType, workerID)
	case mimeType == "text/css":
		l.processCSSFile(task, bodyBytes, mimeType, workerID)
	}
}

// processHTMLFile обрабатывает HTML файл
func (l *Loader) processHTMLFile(task Task, bodyBytes []byte, mimeType string, workerID int) {

	// заменяем ссылки в HTML на локальные
	pageURL, _ := url.Parse(task.URL)
	newBodyBytes, err := linkprocessor.ReplaceLinks(bodyBytes, pageURL, l.baseURL)
	if err != nil {
		fmt.Printf("[воркер %d] ошибка замены ссылок: %v\n", workerID, err)
		return
	}

	// перезаписываем файл с замененными ссылками
	newBodyReader := bytes.NewReader(newBodyBytes)
	localPath, err := filesystem.SaveFile(task.URL, l.baseURL, newBodyReader, mimeType)
	if err != nil {
		fmt.Printf("[воркер %d] ошибка перезаписи файла: %v\n", workerID, err)
		return
	}

	fmt.Printf("[воркер %d] перезаписан с замененными ссылками: %s\n", workerID, localPath)

	// парсим и добавляем новые ссылки, если не превышена глубина
	if task.Depth < l.maxDepth {
		pageLinks, resourceLinks, err := parser.ExtractLinks(pageURL, bytes.NewReader(bodyBytes))
		if err != nil {
			fmt.Printf("[воркер %d] ошибка парсинга HTML: %v\n", workerID, err)
			return
		}

		// добавляем ресурсы (глубина не увеличивается)
		for _, resURL := range resourceLinks {
			l.addTask(resURL, task.Depth, l.getResourceType(resURL))
		}

		// добавляем страницы (глубина +1)
		for _, pageURL := range pageLinks {
			l.addTask(pageURL, task.Depth+1, "html")
		}
	}
}

// processCSSFile обрабатывает CSS файл
func (l *Loader) processCSSFile(task Task, bodyBytes []byte, mimeType string, workerID int) {

	// заменяем ссылки в CSS на локальные
	pageURL, _ := url.Parse(task.URL)
	newBodyBytes, err := linkprocessor.ReplaceCSSLinks(bodyBytes, pageURL, l.baseURL)
	if err != nil {
		fmt.Printf("[воркер %d] ошибка замены ссылок в CSS: %v\n", workerID, err)
		return
	}

	// перезаписываем файл с замененными ссылками
	newBodyReader := bytes.NewReader(newBodyBytes)
	localPath, err := filesystem.SaveFile(task.URL, l.baseURL, newBodyReader, mimeType)
	if err != nil {
		fmt.Printf("[воркер %d] ошибка перезаписи CSS файла: %v\n", workerID, err)
		return
	}

	fmt.Printf("[воркер %d] перезаписан CSS с замененными ссылками: %s\n", workerID, localPath)

	// извлекаем ссылки из CSS и добавляем их как задачи
	cssLinks := parser.ExtractCSSLinks(string(bodyBytes), pageURL)
	for _, cssLink := range cssLinks {
		l.addTask(cssLink, task.Depth, l.getResourceType(cssLink))
	}
}

// addTask добавляет новую задачу в канал
func (l *Loader) addTask(url string, depth int, resType string) {

	if !l.isSameDomain(url) {
		return
	}

	if _, visited := l.visited.Load(url); visited {
		return
	}

	select {
	case l.taskChan <- Task{
		URL:   url,
		Depth: depth,
		Type:  resType}:
		l.taskWg.Add(1)
	case <-l.ctx.Done():
		return
	}
}

// getResourceType определяет тип ресурса по URL
func (l *Loader) getResourceType(urlStr string) string {

	u, err := url.Parse(urlStr)
	if err != nil {
		return "unknown"
	}

	path := strings.ToLower(u.Path)
	switch {
	case strings.HasSuffix(path, ".css"):
		return "css"
	case strings.HasSuffix(path, ".js"):
		return "js"
	case strings.HasSuffix(path, ".png"), strings.HasSuffix(path, ".jpg"),
		strings.HasSuffix(path, ".jpeg"), strings.HasSuffix(path, ".gif"),
		strings.HasSuffix(path, ".svg"), strings.HasSuffix(path, ".webp"),
		strings.HasSuffix(path, ".ico"), strings.HasSuffix(path, ".bmp"):
		return "image"
	case strings.HasSuffix(path, ".woff"), strings.HasSuffix(path, ".woff2"),
		strings.HasSuffix(path, ".ttf"), strings.HasSuffix(path, ".eot"),
		strings.HasSuffix(path, ".otf"):
		return "font"
	case strings.HasSuffix(path, ".html"), strings.HasSuffix(path, ".htm"),
		path == "" || !strings.Contains(filepath.Base(path), "."):
		return "html"
	default:
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
