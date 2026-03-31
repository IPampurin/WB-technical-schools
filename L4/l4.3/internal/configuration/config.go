// общая конфигурация
package configuration

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config - корневой набор настроек для HTTP, БД и поведения календаря
type Config struct {
	HTTP HTTPConfig
	DB   DBConfig
	App  AppConfig
}

// HTTPConfig - порт и таймауты HTTP-сервера
type HTTPConfig struct {
	Host            string        // хост HTTP-сервера
	Port            string        // порт HTTP-сервера
	ReadTimeout     time.Duration // таймаут чтения тела запроса
	WriteTimeout    time.Duration // таймаут записи ответа
	IdleTimeout     time.Duration // keep-alive
	ShutdownTimeout time.Duration // ожидание завершения при graceful shutdown
}

// DBConfig - параметры подключения к Postgres
type DBConfig struct {
	Host     string // хост БД
	Port     int    // порт БД
	User     string // пользователь БД
	Password string // пароль БД
	Name     string // имя БД
}

// DSN - строка подключения для драйвера pgx/database/sql
func (c DBConfig) DSN() string {

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   "/" + c.Name,
	}

	q := url.Values{}
	q.Set("timezone", "UTC")
	u.RawQuery = q.Encode()

	return u.String()
}

// AppConfig - часовой пояс для границ дня/недели/месяца и интервалы фоновых задач
type AppConfig struct {
	Timezone          string        // IANA имя, например "Europe/Moscow" или "UTC"
	ArchiveEvery      time.Duration // частота запуска архивации старых событий
	ReminderQueueSize int           // размер буфера канала напоминальщика
	LogBufferSize     int           // размер буфера асинхронного логгера
}

// Load - читает окружение и подставляет дефолты (ошибка, если Validate не проходит)
func Load() (Config, error) {

	// подставляем дефолты
	cfg := Config{
		HTTP: HTTPConfig{
			Host:            getEnv("HTTP_HOST", "0.0.0.0"),
			Port:            getEnv("HTTP_PORT", "8081"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "postgres"),
			Port:     getInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "postgres"),
		},
		App: AppConfig{
			Timezone:          getEnv("APP_TIMEZONE", "Europe/Moscow"),
			ArchiveEvery:      getDuration("APP_ARCHIVE_EVERY", 5*time.Minute),
			ReminderQueueSize: getInt("APP_REMINDER_QUEUE_SIZE", 10),
			LogBufferSize:     getInt("APP_LOG_BUFFER_SIZE", 10),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate - проверка инвариантов после Load
func (c *Config) Validate() error {

	if c.HTTP.Host == "" {
		return fmt.Errorf("configuration: не задан хост сервера (HTTP_HOST)")
	}
	if c.HTTP.Port == "" {
		return fmt.Errorf("configuration: не задан порт сервера (HTTP_PORT)")
	}
	if _, err := time.LoadLocation(c.App.Timezone); err != nil {
		return fmt.Errorf("configuration: неизвестный часовой пояс APP_TIMEZONE: %w", err)
	}
	if c.DB.Port <= 0 || c.DB.Port > 65535 {
		return fmt.Errorf("configuration: DB_PORT вне допустимого диапазона 1-65535")
	}
	if c.DB.User == "" {
		return fmt.Errorf("configuration: пустой DB_USER")
	}
	if c.DB.Name == "" {
		return fmt.Errorf("configuration: пустой DB_NAME")
	}
	if c.App.ArchiveEvery <= 0 {
		return fmt.Errorf("configuration: APP_ARCHIVE_EVERY должен быть положительным")
	}
	if c.App.ReminderQueueSize < 0 {
		return fmt.Errorf("configuration: APP_REMINDER_QUEUE_SIZE не может быть отрицательным")
	}
	if c.App.LogBufferSize < 0 {
		return fmt.Errorf("configuration: APP_LOG_BUFFER_SIZE не может быть отрицательным")
	}

	return nil
}

// getEnv получает переменную окружения в виде строки
func getEnv(key, def string) string {

	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}

// getInt получает переменную окружения в виде числа
func getInt(key string, def int) int {

	s := os.Getenv(key)
	if s == "" {
		return def
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}

	return n
}

// getDuration получает переменную окружения в виде продолжительности
func getDuration(key string, def time.Duration) time.Duration {

	s := os.Getenv(key)
	if s == "" {
		return def
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}

	return d
}
