## Файлы для GC-MetricsServer - утилита анализа GC и памяти (runtime, профилирование)  

### 📋 Описание проекта и возможности  

**GC-MetricsServer** - это программа-сервис, которая через HTTP-endpoint предоставляет в формате Prometheus  
метрики памяти и сборщика мусора (GC), а также позволяет динамически настраивать процент GC и получать  
профили через pprof.  

### 🖥️ Возможности

**Метрики (Prometheus + JSON):**  
- Используемая память (HeapAlloc) в байтах и человеко-читаемом виде.  
- Общее количество выделенной памяти за всё время.  
- Количество аллокаций (mallocs).  
- Количество завершённых циклов GC.  
- Время последней сборки мусора.  
- Доля CPU, потраченная на GC.  
- Текущее значение GOGC (процент).  

**Управление GOGC:**  
- Чтение текущего процента через GET `/gc_percent`.
- Изменение через POST `/gc_percent?percent=N` (мгновенно применяется).

**Профилирование:**  
- Все стандартные профили `pprof`: heap, goroutine, threadcreate, block, mutex, CPU (требует ручного запуска).

**Веб-интерфейс:**  
- Карточки с метриками, обновляемыми в реальном времени (интервал настраивается).
- Панель управления GOGC: установка нового значения, кнопка обновления.

**Инфраструктура:**  
- Graceful shutdown (SIGINT/SIGTERM).
- Docker + Docker Compose для быстрого старта.
- Асинхронный сбор метрик (без блокировок).

### 📡 Доступные эндпойнты  

  - GET  `/metrics`  Метрики в формате Prometheus 
  - GET  `/api/stats`  JSON со всеми метриками
  - GET  `/api/config`  Конфигурация фронтенда (интервал обновления)
  - GET  `/gc_percent`  Получить текущее значение GOGC
  - POST  `/gc_percent?percent=N`  Установить новый GOGC (N — целое число, например 200)
  - GET  `/debug/pprof/`  Индексная страница pprof
  - GET  `/debug/pprof/heap`  Heap-профиль
  - GET  `/debug/pprof/goroutine`  Профиль горутин
  - GET  `/` Веб-интерфейс
  - GET  `/static/*filepath` Статические файлы (CSS, JS)

### 🗂️ Структура проекта  

``` bash
.
├── cmd/
│   └── main.go                     # точка входа, graceful shutdown
├── internal/
│   ├── collector/
│   │   ├── prometheus.go           # Prometheus collector для runtime.MemStats
│   │   └── runtime.go              # сбор метрик в JSON
│   ├── configuration/
│   │   └── config.go               # загрузка .env и переменных окружения
│   ├── handlers/
│   │   └── metrics.go              # обработчики /api/stats, /gc_percent
│   └── server/
│       ├── routes.go               # регистрация маршрутов (включая pprof)
│       └── server.go               # инициализация Gin, graceful shutdown
├── web/
│   ├── index.html                  # главная страница
│   ├── script.js                   # динамическое обновление, управление GOGC
│   └── style.css                   # стили
├── .env                            # пример переменных окружения
├── compose.yml                     # Docker Compose (сервис + сеть)
├── Dockerfile                      # multi-stage сборка
├── go.mod, go.sum                  # зависимости
└── readme.md                       # этот файл
```

### 🚀 Быстрый старт  

**Требования:**  
- Docker и Docker Compose  
- Свободный порт, указанный в .env (по умолчанию 8080)  

**Запуск:**  

    docker compose up

**После успешного запуска:**  

    Веб-интерфейс:      http://localhost:8080
    Prometheus метрики: http://localhost:8080/metrics
    JSON API:           http://localhost:8080/api/stats
    pprof:              http://localhost:8080/debug/pprof/

Остановка:  

    docker compose down  

Ручной запуск (без Docker):  

    # скопировать .env.example (или создать свой .env)
    cp .env.example .env   # если есть, иначе создать вручную

    # собрать
    go build -o GC-MetricsServer ./cmd/main.go

    # запустить
    ./GC-MetricsServer


### ⚙️ Конфигурация  

Все настройки через файл .env в корне проекта:  

    ## переменные сервера
    HTTP_HOST=0.0.0.0          # хост сервиса
    HTTP_PORT=8080             # порт хоста, на котором работает сервис

    ## переменные сервиса
    METRICS_TIMEOUT=3s         # частота сбора метрик

### 📊 Примеры запросов  

1. Получить метрики в Prometheus-формате

```bash
curl http://localhost:8080/metrics
```

Пример ответа:  

    # HELP go_memstats_alloc_bytes Количество байт, выделенных и используемых в данный момент (HeapAlloc)
    # TYPE go_memstats_alloc_bytes gauge
    go_memstats_alloc_bytes 2.345e+06
    # HELP go_gc_num_gc Количество завершённых циклов GC
    # TYPE go_gc_num_gc counter
    go_gc_num_gc 42
    ...

2. Получить JSON-метрики  

```bash
curl http://localhost:8080/api/stats
```

Ответ:  

```json
{
  "alloc_bytes": "2.34 MB",
  "total_alloc_bytes": "123.45 MB",
  "mallocs": 125600,
  "num_gc": 42,
  "last_gc_time": "2026-04-04 15:04:05.123",
  "gc_cpu_fraction": 0.0012,
  "gc_percent": 100
}
```

3. Узнать текущий GOGC  

```bash
curl http://localhost:8080/gc_percent
```
```json
{"gc_percent":100}
```

4. Изменить GOGC на 200  

```bash
curl -X POST "http://localhost:8080/gc_percent?percent=200"
```
```json
{"new_percent":200,"old_percent":100}
```

5. Получить heap-профиль (pprof)

```bash
curl -o heap.out http://localhost:8080/debug/pprof/heap
go tool pprof -http=:8082 heap.out
```

### 📦 Метрики  

Сервис экспортирует следующие Prometheus-метрики:  

    Метрика                       Тип	    Описание  
    go_memstats_alloc_bytes       gauge     Текущее количество выделенной и используемой памяти (HeapAlloc)
    go_memstats_total_alloc_bytes counter   Общее количество выделенной памяти за всё время
    go_memstats_mallocs_total     counter   Общее количество операций выделения памяти (mallocs)
    go_gc_num_gc                  counter   Количество завершённых циклов сборки мусора
    go_gc_last_gc_time_seconds    gauge     Unix timestamp последнего GC (в секундах)
    go_gc_cpu_fraction            gauge     Доля процессорного времени, потраченная на GC с момента старта
    go_gc_percent                 gauge     Текущее значение GOGC (процент, при котором запускается GC)

### 📦 Зависимости  

- gin-gonic/gin - HTTP-роутер  
- prometheus/client_golang - экспорт метрик Prometheus  

Устанавливаются через go mod.  

