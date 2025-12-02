package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/redis/go-redis/v9"
)

// выносим константы конфигурации по умолчанию, чтобы были на виду
const (
	redisPortConst     = "6379" // порт, на котором сидит рэдис по умолчанию
	redisPasswordConst = ""     // пароль от БД рэдиса по умолчанию
	redisDBNumberConst = 0      // номер БД рэдиса по умолчанию
	redisTTLConst      = 600    // время жизни данных в кэше в секундах по умолчанию
	thresholdConst     = 24     // время в часах за которое берём записи для прогрева кэша
)

// CacheConfig описывает настройки с учётом переменных окружения
type CacheConfig struct {
	RedisPort     string        // порт, на котором сидит рэдис
	RedisPassword string        // пароль от БД рэдиса
	RedisDBNumber int           // номер БД рэдиса
	RedisTTL      time.Duration // время жизни данных в кэше в секундах
}

var (
	config atomic.Value  // атомарное хранилище для конфигурации
	Rdb    *redis.Client // клиент Redis
)

// getEnvString проверяет наличие и корректность переменной окружения (строковое значение)
func getEnvString(envVariable, defaultValue string) string {

	value, ok := os.LookupEnv(envVariable)
	if ok {
		return value
	}

	return defaultValue
}

// getEnvInt проверяет наличие и корректность переменной окружения (числовое значение > 0)
func getEnvInt(envVariable string, defaultValue int) int {

	value, ok := os.LookupEnv(envVariable)
	if ok {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
		log.Printf("ошибка парсинга %s, используем значение по умолчанию: %d", envVariable, defaultValue)
	}

	return defaultValue
}

// readConfig уточняет конфигурацию с учётом переменных окружения
func readConfig() *CacheConfig {

	return &CacheConfig{
		RedisPort:     getEnvString("REDIS_PORT", redisPortConst),
		RedisPassword: getEnvString("REDIS_PASSWORD", redisPasswordConst),
		RedisDBNumber: getEnvInt("REDIS_DB", redisDBNumberConst),
		RedisTTL:      time.Duration(getEnvInt("REDIS_TTL", redisTTLConst)) * time.Second,
	}
}

// getConfig безопасно получает конфигурацию
func getConfig() *CacheConfig {

	if cfg := config.Load(); cfg != nil {
		return cfg.(*CacheConfig)
	}
	// возвращаем конфигурацию по умолчанию, если ещё не инициализировано
	return &CacheConfig{
		RedisPort:     redisPortConst,
		RedisPassword: redisPasswordConst,
		RedisDBNumber: redisDBNumberConst,
		RedisTTL:      time.Duration(redisTTLConst) * time.Second,
	}
}

// updateConfig обновляет конфигурацию (для hot reload в будущем)
func updateConfig(newConfig *CacheConfig) {

	config.Store(newConfig)
}

// InitRedis запускает работу с Redis
func InitRedis() error {

	// считываем конфигурацию
	cfg := readConfig()

	// проверяем номер базы в рэдисе
	if cfg.RedisDBNumber < 0 || 16 < cfg.RedisDBNumber {
		log.Printf("Проверьте .env файл, ошибка назначения базы данных Redis. Ожидается значение: 0 ... 16. Получено: %d\n", cfg.RedisDBNumber)
		log.Printf("Используется значение по умолчанию: %d\n", redisDBNumberConst)
		cfg.RedisDBNumber = redisDBNumberConst
	}

	// проверяем время жизни данных в кэше
	if cfg.RedisTTL < 0 {
		log.Printf("Проверьте .env файл, ошибка назначения TTL для Redis. Ожидается положительное значение. Получено: %d\n", cfg.RedisTTL)
		log.Printf("Используется значение по умолчанию: %dс.\n", redisTTLConst)
		cfg.RedisTTL = time.Duration(redisTTLConst) * time.Second
	}

	// атомарно сохраняем конфигурацию
	updateConfig(cfg)

	// заводим клиента Redis
	Rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", "redis", cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDBNumber,
	})

	log.Println("Клиент Redis запущен.")

	// проверяем подключение
	if err := Rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("ошибка подключения к Redis: %v\n", err)
		return err
	}

	log.Println("Подключение к Redis есть.")
	log.Println("Начинаем загрузку первичных данных в кэш.")

	// загружаем начальные данные
	err := loadDataToCache()
	if err != nil {
		log.Printf("ошибка загрузки первичных данных в кэш: %v", err)
		return err
	}

	log.Println("Загрузка первичных данных в кэш завершена.")

	return nil
}

// GetTTL определяет время жизни данных в кэше (для postOrder понадобится)
func GetTTL() time.Duration {

	return getConfig().RedisTTL
}

// loadDataToCache загружает данные за последнее время в кэш при старте
func loadDataToCache() error {

	// устанавливаем временной порог (например, 24 часа)
	timeThreshold := time.Now().Add(-time.Duration(thresholdConst) * time.Hour)

	// получаем заказы до установленного порога
	var orders []models.Order
	err := db.DB.Db.
		Preload("Delivery").
		Preload("Payment").
		Preload("Items").
		Where("date_created >= ?", timeThreshold).
		Find(&orders).Error
	if err != nil {
		return fmt.Errorf("ошибка при получении заказов: %v", err)
	}

	log.Printf("Найдено %d заказов за последние %d часа", len(orders), thresholdConst)

	// сохраняем данные в redis
	keyValues := make(map[string]interface{})
	for _, order := range orders {
		keyValues[fmt.Sprintf("order:%s", order.OrderUID)] = order
	}
	if err := BatchSet(keyValues); err != nil {
		log.Printf("Ошибка группового кэширования при старте: %v", err)
	}

	return nil
}

// GetCache получает запись из кэша
func GetCache(key string) ([]byte, error) {

	if Rdb == nil {
		return nil, fmt.Errorf("ошибка при получении записи из кэша: Redis клиент не инициализирован")
	}

	return Rdb.Get(context.Background(), key).Bytes()
}

// SetCache сохраняет запись в кэш
func SetCache(key string, value interface{}) error {

	if Rdb == nil {
		return fmt.Errorf("ошибка при сохранении записи в кэш: Redis клиент не инициализирован")
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("ошибка сериализации данных при добавлении в кэш: %w", err)
	}

	return Rdb.Set(context.Background(), key, jsonData, GetTTL()).Err()
}

// DelCache удаляет запись из кэша
func DelCache(key string) error {

	if Rdb == nil {
		return fmt.Errorf("ошибка при удалении записи из кэша: Redis клиент не инициализирован")
	}

	return Rdb.Del(context.Background(), key).Err()
}

// BatchGetKeys проверяет существование нескольких ключей в кэше за один запрос, возвращает map[ключ]существует ли
func BatchGetKeys(keys []string) (map[string]bool, error) {

	if Rdb == nil {
		return nil, fmt.Errorf("ошибка BatchGetKeys при получении записи из кэша: Redis клиент не инициализирован")
	}

	ctx := context.Background()

	result := make(map[string]bool)
	if len(keys) == 0 {
		return result, nil
	}

	// используем pipeline для отправки всех команд разом
	pipe := Rdb.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))

	// подготавливаем команды Exists для каждого ключа
	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}

	// выполняем все команды разом
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения pipeline запроса в кэше: %w", err)
	}

	// обрабатываем результаты
	for i, cmd := range cmds {
		count, err := cmd.Result()
		if err != nil {
			// логируем ошибку, но продолжаем обработку других ключей
			log.Printf("Ошибка проверки ключа в кэше %s: %v", keys[i], err)
			result[keys[i]] = false
		} else {
			result[keys[i]] = count > 0
		}
	}

	return result, nil
}

// BatchSet сохраняет несколько записей в кэш за один запрос
func BatchSet(keyValues map[string]interface{}) error {

	if Rdb == nil {
		return fmt.Errorf("ошибка BatchSet при сохранении записи в кэш: Redis клиент не инициализирован")
	}

	ctx := context.Background()

	if len(keyValues) == 0 {
		return nil
	}

	pipe := Rdb.Pipeline()

	// подготавливаем команды SET для каждой пары ключ-значение
	for key, value := range keyValues {
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("ошибка сериализации в кэше для ключа %s: %w", key, err)
		}
		pipe.Set(ctx, key, jsonData, GetTTL())
	}

	// выполняем все команды разом
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("ошибка выполнения pipeline для множественной записи в кэше: %w", err)
	}

	return nil
}

// BatchGet получает значения нескольких ключей (если нужно не только проверять существование)
func BatchGet(keys []string) (map[string][]byte, error) {

	if Rdb == nil {
		return nil, fmt.Errorf("ошибка BatchGet при получении записи из кэша: Redis клиент не инициализирован")
	}

	ctx := context.Background()

	result := make(map[string][]byte)
	if len(keys) == 0 {
		return result, nil
	}

	pipe := Rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	// подготавливаем команды GET для каждого ключа
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	// выполняем все команды разом
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		// игнорируем redis.Nil (ключ не найден) - это нормально
		return nil, fmt.Errorf("ошибка выполнения pipeline запроса в кэше: %w", err)
	}

	// обрабатываем результаты
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == nil {
			result[keys[i]] = []byte(val)
		} else if !errors.Is(err, redis.Nil) {
			// логируем только настоящие ошибки (не "ключ не найден")
			log.Printf("Ошибка получения ключа в кэше %s: %v", keys[i], err)
		}
	}

	return result, nil
}
