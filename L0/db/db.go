package db

import (
	"fmt"
	"log"
	"os"

	"github.com/IPampurin/WB-technical-schools/L0/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Dbinstance struct {
	Db *gorm.DB
}

var DB Dbinstance

func ConnectDB() error {

	// dsn - URL для соединения с базой данных.
	// Имя пользователя базы данных, пароль и имя базы данных, а также порт базы берутся из
	// переменных окружения, они описаны в файле .env
	dsn := fmt.Sprintf(
		"host=db user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Moscow",
		os.Getenv("DBL0_USER"),
		os.Getenv("DBL0_PASSWORD"),
		os.Getenv("DBL0_NAME"),
		os.Getenv("DBL0_PORT"),
	)
	// создаём подключение к базе данных.
	// В &gorm.Config настраивается логер, который будет сохранять информацию
	// обо всех активностях с базой данных.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Printf("Не удалось подключиться к базе данных: %v", err)
		return fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	log.Println("Подключение к базе данных установлено.")
	db.Logger = logger.Default.LogMode(logger.Info)

	log.Println("Запуск миграций.")

	// собираем модели в срез
	models := []interface{}{
		&models.Order{},
		&models.Delivery{},
		&models.Payment{},
		&models.Item{},
	}
	// автомиграцию моделей выполняем списком
	err = db.AutoMigrate(models...)
	if err != nil {
		log.Printf("Ошибка при выполнении миграций: %v", err)
		return fmt.Errorf("ошибка миграции: %w", err)
	}

	log.Println("Миграции успешно применены.")

	DB = Dbinstance{
		Db: db,
	}

	return nil
}
