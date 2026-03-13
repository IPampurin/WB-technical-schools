package configuration

import (
	"time"

	cleanenvport "github.com/wb-go/wbf/config/cleanenv-port"
)

// ConfServer — параметры HTTP-сервера
type ConfServer struct {
	HostName string `env:"SERVICE_HOST_NAME"  env-default:"localhost"`
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

type ConfBroker struct {
	HostName      string        `env:"RABBIT_HOST_NAME"        env-default:"rabbitMQ"`
	Port          int           `env:"RABBIT_PORT"             env-default:"5672"`
	User          string        `env:"RABBIT_USER"             env-default:"rabbitMQ"`
	Password      string        `env:"RABBIT_PASSWORD"         env-default:"rabbit"`
	VHost         string        `env:"RABBIT_VHOST"            env-default:"/"`
	Queue         string        `env:"RABBIT_QUEUE"            env-default:"bookingQueue"`
	Exchange      string        `env:"RABBIT_EXCHANGE"         env-default:"bookingExchange"`
	RoutingKey    string        `env:"RABBIT_ROUTING_KEY"      env-default:"booking"`
	ConnName      string        `env:"RABBIT_CONNECTION_NAME"  env-default:"eventbooker"`
	ConnTimeout   time.Duration `env:"RABBIT_CONNECT_TIMEOUT"  env-default:"5s"`
	Heartbeat     time.Duration `env:"RABBIT_HEARTBEAT"        env-default:"10s"`
	RetryAttempts int           `env:"RABBIT_RETRY_ATTEMPTS"   env-default:"3"`
	RetryDelay    time.Duration `env:"RABBIT_RETRY_DELAY"      env-default:"3s"`
	RetryBackoff  float64       `env:"RABBIT_RETRY_BACKOFF"    env-default:"2.0"`
}

// ConfZSet — параметры Redis
type ConfZSet struct {
	HostName          string        `env:"REDIS_HOST_NAME"     env-default:"dbRedis"`
	Port              int           `env:"REDIS_PORT"          env-default:"6379"`
	Password          string        `env:"REDIS_PASSWORD"      env-default:""`
	DB                int           `env:"REDIS_DB"            env-default:"0"`
	OverdueKey        string        `env:"ZSET_OVERDUE_KEY"    env-default:"overdue:bookings"`
	CheckInterval     time.Duration `env:"ZSET_CHECK_INTERVAL" env-default:"1s"`
	ReadRetryAttempts int           `env:"REDIS_READ_RETRY_ATTEMPTS" env-default:"3"`
	ReadRetryDelay    time.Duration `env:"REDIS_READ_RETRY_DELAY"    env-default:"100ms"`
	ReadRetryBackoff  float64       `env:"REDIS_READ_RETRY_BACKOFF"  env-default:"2.0"`
}

// Config — корневая структура конфигурации
type Config struct {
	Server ConfServer
	DB     ConfDB
	Broker ConfBroker
	ZSet   ConfZSet
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
