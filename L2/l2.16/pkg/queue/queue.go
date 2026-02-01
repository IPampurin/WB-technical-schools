package queue

import "sync"

// Task - задача для загрузки
type Task struct {
	URL   string
	Depth int
	Type  string // "html", "css", "js", "image", "font"
}

// Queue - потокобезопасная очередь задач
type Queue struct {
	tasks []Task
	mu    sync.Mutex
}

// New создает новую очередь
func New() *Queue {

	return &Queue{
		tasks: make([]Task, 0),
		mu:    sync.Mutex{}, // можно и не указывать, инициализируется автоматически
	}
}

// Push добавляет задачу в конец очереди
func (q *Queue) Push(task Task) {

	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks = append(q.tasks, task)
}

// Pop извлекает задачу из начала очереди
func (q *Queue) Pop() (Task, bool) {

	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tasks) == 0 {
		return Task{}, false
	}

	task := q.tasks[0]
	q.tasks = q.tasks[1:]

	return task, true
}

// IsEmpty проверяет, пуста ли очередь
func (q *Queue) IsEmpty() bool {

	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.tasks) == 0
}
