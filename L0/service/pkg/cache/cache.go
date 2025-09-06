package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client // клиент Redis

// InitRedis запускает работу с Redis
func InitRedis() error {

	// смотрим переменные окружения
	portRedis, ok := os.LookupEnv("REDIS_PORT")
	if !ok {
		portRedis = "6379"
	}
	dbRedisStr, ok := os.LookupEnv("REDIS_DB")
	if !ok {
		dbRedisStr = "0"
	}
	passwordRedis, ok := os.LookupEnv("REDIS_PASSWORD")
	if !ok {
		passwordRedis = ""
	}

	// переведём в int номер базы в Redis
	dbRedis, err := strconv.Atoi(dbRedisStr)
	if err != nil || dbRedis < 0 || dbRedis > 16 {
		log.Printf("проверьте .env файл, ошибка назначения базы данных Redis: %v\n", err)
		return err
	}

	// узнаем время жизни данных в кэше
	ttl := GetTTL()

	// заводим клиента Redis
	Rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", "redis", portRedis),
		Password: passwordRedis,
		DB:       dbRedis,
	})

	log.Println("Клиент Redis запущен.")

	// проверяем подключение
	if err = Rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("ошибка подключения к Redis: %v\n", err)
		return err
	}

	log.Println("Подключение к Redis есть.")
	log.Println("Начинаем загрузку первичных данных в кэш.")

	// загружаем начальные данные
	err = loadDataToCache(time.Duration(ttl) * time.Second)
	if err != nil {
		log.Printf("ошибка загрузки первичных данных в кэш: %v", err)
		return err
	}

	log.Println("Загрузка первичных данных в кэш завершена.")

	return nil
}

// GetTTL определяет время жизни данных в кэше
func GetTTL() time.Duration {

	// смотрим переменную окружения
	ttlStr, ok := os.LookupEnv("REDIS_TTL")
	if !ok {
		ttlStr = "600" // по умолчанию примем 10 минут
	}

	ttl, err := strconv.Atoi(ttlStr)
	if err != nil || ttl < 0 {
		log.Printf("Проверьте .env файл, ошибка назначения времени жизни данных в Redis. Используется значение по умолчанию.")
		return 600 * time.Second
	}

	return time.Duration(ttl) * time.Second
}

// loadDataToCache загружает данные за последнее время в кэш при старте
func loadDataToCache(ttl time.Duration) error {

	// по умочанию установим временной порог, например, 24 часа
	threshold := 24

	timeThreshold := time.Now().Add(-time.Duration(threshold) * time.Hour)

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

	log.Printf("Найдено %d заказов за последние %v часа", len(orders), threshold)

	// сохраняем данные в redis
	for _, order := range orders {
		if err := SetCahe(fmt.Sprintf("order:%s", order.OrderUID), order, ttl); err != nil {
			log.Printf("ошибка кэширования заказа %s при старте: %v", order.OrderUID, err)
		}
	}

	return nil
}

// GetCache получает запись из кэша
func GetCache(key string) ([]byte, error) {

	return Rdb.Get(context.Background(), key).Bytes()
}

// SetCahe сохраняет запись в кэш
func SetCahe(key string, value interface{}, ttl time.Duration) error {

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("ошибка сериализации данных при добавлении в кэш: %w", err)
	}
	return Rdb.Set(context.Background(), key, jsonData, ttl).Err()
}

// DelCache удаляет запись из кэша
func DelCache(key string) error {

	return Rdb.Del(context.Background(), key).Err()
}
