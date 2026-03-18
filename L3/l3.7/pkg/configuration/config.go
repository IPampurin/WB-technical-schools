package configuration

import (
	cleanenvport "github.com/wb-go/wbf/config/cleanenv-port"
)

// ConfServer - параметры HTTP-сервера
type ConfServer struct {
	HostName string `env:"SERVICE_HOST_NAME"  env-default:"0.0.0.0"`
	Port     int    `env:"SERVICE_PORT"       env-default:"8081"`
	GinMode  string `env:"GIN_MODE"           env-default:"debug"`
}

// ConfDB - параметры подключения к PostgreSQL
type ConfDB struct {
	HostName string `env:"DB_HOST_NAME" env-default:"postgres"`
	Port     int    `env:"DB_PORT"      env-default:"5432"`
	Name     string `env:"DB_NAME"      env-default:"postgres"`
	User     string `env:"DB_USER"      env-default:"postgres"`
	Password string `env:"DB_PASSWORD"  env-default:"postgres"`
}

// ConfAuth - ключ подписи jwt
type ConfAuth struct {
	SecretKey string `env:"SECRET_KEY_JWT" env-default:""`
}

// Config - корневая структура конфигурации
type Config struct {
	Server  ConfServer
	DB      ConfDB
	AuthKey ConfAuth
}

// ReadConfig загружает .env файл из корня проекта и возвращает заполненную структуру Config
func ReadConfig() (*Config, error) {

	var config Config

	// загружаем конфигурацию из файла .env напрямую в структуру
	if err := cleanenvport.LoadPath("./.env", &config); err != nil {
		return nil, err
	}

	return &config, nil
}
