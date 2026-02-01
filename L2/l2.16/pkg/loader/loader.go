package loader

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"mywget/pkg/queue"
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
		wg:        &sync.WaitGroup{},
	}

	// 3. запускаем загрузчик
	return loader.run()
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

	// TODO: Здесь будет реальная загрузка
	// временная заглушка
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("[Воркер %d] Загружено: %s\n", workerID, task.URL)
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
