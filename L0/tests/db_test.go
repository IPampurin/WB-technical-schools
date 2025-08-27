package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/IPampurin/WB-technical-schools/L0/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Функция для подсчета записей
func countOrders(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&models.Order{}).Count(&count).Error
	return count, err
}

// Функция открытия БД с реальными параметрами подключения
func openTestDB(t *testing.T) *gorm.DB {
	// Получаем параметры подключения из окружения
	portDB, ok := os.LookupEnv("DBL0_PORT")
	if !ok {
		portDB = "5432"
	}
	nameDB, ok := os.LookupEnv("DBL0_NAME")
	if !ok {
		nameDB = "level_zero_db"
	}
	passwordDB, ok := os.LookupEnv("DBL0_PASSWORD")
	if !ok {
		passwordDB = "postgres"
	}
	userDB, ok := os.LookupEnv("DBL0_USER")
	if !ok {
		userDB = "postgres"
	}

	// Формируем DSN с правильным хостом и параметрами
	dsn := fmt.Sprintf(
		"host=localhost user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Moscow",
		userDB,
		passwordDB,
		nameDB,
		portDB,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)

	// Создаем таблицы
	err = db.AutoMigrate(
		&models.Order{},
		&models.Delivery{},
		&models.Payment{},
		&models.Item{},
	)
	assert.NoError(t, err)

	return db
}

// TestDBOperations тестирует взаимодействие с нашей базой данных в зпущенном состоянии
func TestDBOperations(t *testing.T) {
	// Подключаемся к БД
	testDB := openTestDB(t)

	// Получаем объект *sql.DB для закрытия соединения
	sqlDB, err := testDB.DB()
	assert.NoError(t, err)
	defer sqlDB.Close()

	// Сохраняем начальное количество записей
	before, err := countOrders(testDB)
	assert.NoError(t, err)

	// Создаем тестовый заказ
	testOrder := models.Order{
		OrderUID:        "test_order_uid",
		TrackNumber:     "test_track",
		Entry:           "test_entry",
		Locale:          "ru",
		CustomerID:      "test_customer",
		DeliveryService: "test_service",
		Shardkey:        "1",
		SMID:            1,
		Delivery: models.Delivery{
			Name:    "Test User",
			Phone:   "+79991234567",
			City:    "Москва",
			Address: "ул. Тестовая, 1",
		},
		Payment: models.Payment{
			Transaction: "test_transaction",
			Currency:    "RUB",
			Provider:    "test_provider",
			Amount:      1000.0,
		},
	}

	// Сохраняем заказ
	err = testDB.Create(&testOrder).Error
	assert.NoError(t, err)

	// Проверяем сохранение
	var savedOrder models.Order
	err = testDB.First(&savedOrder, testOrder.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, testOrder.OrderUID, savedOrder.OrderUID)
	assert.Equal(t, testOrder.TrackNumber, savedOrder.TrackNumber)

	// Удаляем запись
	err = testDB.Delete(&models.Order{}, testOrder.ID).Error
	assert.NoError(t, err)

	// Проверяем количество записей
	after, err := countOrders(testDB)
	assert.NoError(t, err)
	assert.Equal(t, before, after)
}
