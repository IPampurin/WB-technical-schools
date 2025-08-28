package db

import (
	"fmt"
	"log"
	"os"

	"github.com/IPampurin/WB-technical-schools/L0/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Dbinstance struct {
	Db *gorm.DB
}

var DB Dbinstance

// ConnectDB устанавливает соединение с базой данных
func ConnectDB() error {

	// Имя пользователя базы данных, пароль и имя базы данных, а также порт базы берутся из
	// переменных окружения, рекомендуется описать их в файле .env
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

	// dsn - URL для соединения с базой данных.
	dsn := fmt.Sprintf(
		"host=db user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Moscow",
		userDB,
		passwordDB,
		nameDB,
		portDB,
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

// CloseDB закрывает соединение с базой
func CloseDB() {

	// получаем объект *sql.DB для закрытия соединения
	sqlDB, err := DB.Db.DB()
	if err != nil {
		log.Printf("Ошибка при получении SQL соединения: %v", err)
		return
	}

	// закрываем соединение
	if err := sqlDB.Close(); err != nil {
		log.Printf("Предупреждение: ошибка при закрытии БД: %v", err)
	} else {
		log.Println("БД успешно отключена.")
	}
}
