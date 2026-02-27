package configuration

import (
	"time"

	cleanenvport "github.com/wb-go/wbf/config/cleanenv-port"
)

// ConfServer — параметры HTTP-сервера
type ConfServer struct {
	HostName string `env:"SERVICE_HOST_NAME" env-default:"localhost"`
	Port     int    `env:"SERVICE_PORT"       env-default:"8081"`
	GinMode  string `env:"GIN_MODE"           env-default:"debug"`
}

// ConfDB — параметры подключения к PostgreSQL
type ConfDB struct {
	HostName string `env:"DB_HOST_NAME" env-default:"dbPostgres"`
	Port     int    `env:"DB_PORT"      env-default:"5432"`
	Name     string `env:"DB_NAME"      env-default:"db-postgres"`
	User     string `env:"DB_USER"      env-default:"postgres"`
	Password string `env:"DB_PASSWORD"  env-default:"postgres"`
}

// ConfCache — параметры Redis
type ConfCache struct {
	HostName string        `env:"REDIS_HOST_NAME" env-default:"dbRedis"`
	Port     int           `env:"REDIS_PORT"      env-default:"6379"`
	Password string        `env:"REDIS_PASSWORD"  env-default:""`
	DB       int           `env:"REDIS_DB"        env-default:"0"`
	TTL      time.Duration `env:"REDIS_TTL"       env-default:"600s"`
	Warming  time.Duration `env:"REDIS_WARMING"   env-default:"24h"`
}

// Config — корневая структура конфигурации
type Config struct {
	Server ConfServer
	DB     ConfDB
	Redis  ConfCache
}

// ReadConfig загружает .env файл из корня проекта и возвращает заполненную структуру Config
func ReadConfig() (*Config, error) {

	var config Config

	// загружаем конфигурацию из файла .env напрямую в структуру
	if err := cleanenvport.LoadPath("./.env", &config); err != nil {
		return nil, err
	}

	// дополнительной обработки для time.Duration больше не требуется,
	// так как мы указали единицы измерения прямо в теге env-default (например, "600s", "100ms", "60s")

	return &config, nil
}
