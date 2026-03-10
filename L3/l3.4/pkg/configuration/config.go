package configuration

import (
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

// ConfS3 - параметры подключения к внешнему S3 хранилищу
type ConfS3 struct {
	Endpoint  string `env:"S3_ENDPOINT"   env-default:"localhost:9000"`
	AccessKey string `env:"S3_ACCESS_KEY" env-default:"minioadmin"`
	SecretKey string `env:"S3_SECRET_KEY" env-default:"minioadmin"`
	Bucket    string `env:"S3_BUCKET"     env-default:"images"`
	UseSSL    bool   `env:"S3_USE_SSL"    env-default:"false"`
}

// ConfKafka - параметры работы брокера
type ConfKafka struct {
	HostName      string `env:"KAFKA_HOST_NAME"     env-default:"localhost"`
	Port          int    `env:"KAFKA_PORT_NUM"      env-default:"9092"`
	InputTopic    string `env:"KAFKA_INPUT_TOPIC"   env-default:"image-tasks"`
	OutputTopic   string `env:"KAFKA_OUTPUT_TOPIC"  env-default:"image-results"`
	ConsumerGroup string `env:"KAFKA_CONSUMER_GROUP" env-default:"image-processor-manager"`
}

// ConfThumb - параметры миниатюр
type ConfThumb struct {
	ThumbWidth  int `env:"WIDTH_THUMBNAIL"  env-default:"150"`
	ThumbHeight int `env:"HEIGHT_THUMBNAIL" env-default:"150"`
}

// Config — корневая структура конфигурации
type Config struct {
	Server ConfServer
	DB     ConfDB
	S3     ConfS3
	Kafka  ConfKafka
	Thumb  ConfThumb
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
