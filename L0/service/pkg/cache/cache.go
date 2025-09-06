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
	ttlStr, ok := os.LookupEnv("REDIS_TTL")
	if !ok {
		ttlStr = "600"
	}

	// переведём в int номер базы в Redis и время жизни данных
	dbRedis, err := strconv.Atoi(dbRedisStr)
	if err != nil {
		log.Printf("проверьте .env файл, ошибка назначения базы данных Redis: %v\n", err)
		return err
	}
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil {
		log.Printf("проверьте .env файл, ошибка назначения времени жизни данных в Redis: %v\n", err)
		return err
	}

	// заводим клиента Redis
	Rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", "localhost", portRedis),
		Password: passwordRedis,
		DB:       dbRedis,
	})

	// проверяем подключение
	if err = Rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("ошибка подключения к Redis: %v\n", err)
		return err
	}

	// загружаем начальные данные
	err = loadDataToCache(time.Duration(ttl) * time.Second)
	if err != nil {
		log.Printf("ошибка загрузки первичных данных в кэш: %v", err)
		return err
	}

	return nil
}

// loadDataToCache загружает данные за крайние 24 часа в кэш при старте
func loadDataToCache(ttl time.Duration) error {

	timeThreshold := time.Now().Add(-24 * time.Hour) // временной порог 24 часа

	// получаем заказы за крайние 24 часа
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

	log.Printf("Найдено %d заказов за последние 24 часа", len(orders))

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
