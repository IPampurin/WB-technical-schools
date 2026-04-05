package configuration

import (
	"fmt"
	"os"
	"time"
)

// Config хранит настройки сервера и сбора метрик
type Config struct {
	Host           string        // хост для HTTP-сервера
	Port           string        // порт
	MetricsTimeout time.Duration // интервал обновления метрик для фронтенда
}

// Load загружает конфигурацию из .env и переменных окружения
func Load() (*Config, error) {

	// подставляем дефолты
	cfg := &Config{
		Host:           getEnv("HTTP_HOST", "0.0.0.0"),
		Port:           getEnv("HTTP_PORT", "8080"),
		MetricsTimeout: getDuration("METRICS_TIMEOUT", 3*time.Second),
	}

	if err := cfg.Validate(); err != nil {
		return &Config{}, err
	}

	return cfg, nil
}

// Validate - проверка инвариантов после Load
func (c *Config) Validate() error {

	if c.Host == "" {
		return fmt.Errorf("configuration: не задан хост сервера (HTTP_HOST)")
	}
	if c.Port == "" {
		return fmt.Errorf("configuration: не задан порт сервера (HTTP_PORT)")
	}
	if c.MetricsTimeout <= 0 {
		return fmt.Errorf("configuration: METRICS_TIMEOUT должен быть положительным")
	}

	return nil
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
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
