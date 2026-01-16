package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
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
		nameDB = "db-postgres"
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

	// Проверяем подключение
	err = db.Exec("SELECT 1").Error
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

	// Создаем уникальный тестовый заказ с timestamp
	timestamp := time.Now().Unix()
	testOrder := models.Order{
		OrderUID:        fmt.Sprintf("test_order_uid_%d", timestamp), // Уникальный ID
		TrackNumber:     fmt.Sprintf("test_track_%d", timestamp),
		Entry:           "test_entry",
		Locale:          "ru",
		CustomerID:      "test_customer",
		DeliveryService: "test_service",
		Shardkey:        "1",
		SMID:            1,
		Delivery: models.Delivery{
			Name:    "Test User",
			Phone:   "+79991234567",
			Zip:     "123456",
			City:    "Москва",
			Address: "ул. Тестовая, 1",
			Region:  "Московская область",
			Email:   fmt.Sprintf("test%d@example.com", timestamp),
		},
		Payment: models.Payment{
			Transaction:  fmt.Sprintf("test_transaction_%d", timestamp), // Уникальный transaction
			Currency:     "RUB",
			Provider:     "test_provider",
			Amount:       1000.0,
			PaymentDT:    timestamp,
			Bank:         "test_bank",
			DeliveryCost: 100.0,
			GoodsTotal:   900.0,
			CustomFee:    0.0,
		},
		Items: []models.Item{
			{
				ChrtID:      int(timestamp),
				TrackNumber: fmt.Sprintf("item_track_%d", timestamp),
				Price:       500.0,
				RID:         fmt.Sprintf("rid_%d", timestamp),
				Name:        "Test Item",
				Sale:        0.0,
				Size:        "M",
				TotalPrice:  500.0,
				NMID:        int(timestamp),
				Brand:       "Test Brand",
				Status:      1,
			},
		},
	}

	// Сохраняем заказ
	err = testDB.Create(&testOrder).Error
	assert.NoError(t, err)

	// Проверяем сохранение - ищем по OrderUID, а не ID
	var savedOrder models.Order
	err = testDB.Preload("Delivery").Preload("Payment").Preload("Items").
		Where("order_uid = ?", testOrder.OrderUID).
		First(&savedOrder).Error
	assert.NoError(t, err)
	assert.Equal(t, testOrder.OrderUID, savedOrder.OrderUID)
	assert.Equal(t, testOrder.TrackNumber, savedOrder.TrackNumber)
	assert.Len(t, savedOrder.Items, 1)

	// Удаляем запись и связанные данные
	err = testDB.Where("order_uid = ?", testOrder.OrderUID).Delete(&models.Order{}).Error
	assert.NoError(t, err)

	// Проверяем количество записей
	after, err := countOrders(testDB)
	assert.NoError(t, err)
	assert.Equal(t, before, after)
}
