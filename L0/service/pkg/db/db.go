package db

import (
	"fmt"
	"log"
	"os"

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
		nameDB = "level-zero-db"
	}
	passwordDB, ok := os.LookupEnv("DBL0_PASSWORD")
	if !ok {
		passwordDB = "postgres"
	}
	userDB, ok := os.LookupEnv("DBL0_USER")
	if !ok {
		userDB = "postgres"
	}

	// dsn - URL для соединения с базой данных. db имя сервиса БД из docker-compose
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

	// выполняем миграцию моделей
	err = runMigrations(db)
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

// runMigrations выполняет последовательное создание всех таблиц базы данных
// таблицы создаются в порядке зависимостей: сначала родительские, затем дочерние
func runMigrations(db *gorm.DB) error {
	// Создаем таблицы в правильном порядке с учетом зависимостей
	migrations := []func(*gorm.DB) error{
		createOrdersTable,     // основная таблица заказов (родительская)
		createDeliveriesTable, // таблица доставки (зависит от orders)
		createPaymentsTable,   // таблица платежей (зависит от orders)
		createItemsTable,      // таблица товаров (зависит от orders)
	}

	// выполняем каждую миграцию последовательно
	for _, migration := range migrations {
		if err := migration(db); err != nil {
			return err
		}
	}

	return nil
}

// createOrdersTable создает таблицу orders с основными полями
// эта таблица является корневой для всей структуры данных заказа
func createOrdersTable(db *gorm.DB) error {
	sql := `
		-- создаем основную таблицу заказов, если она не существует
		CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,                            -- автоинкрементный идентификатор
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,   -- метка времени создания записи (gorm.Model)
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,   -- метка времени обновления записи (gorm.Model)
			deleted_at TIMESTAMP NULL,                        -- мягкое удаление (gorm.Model)
			order_uid VARCHAR(255) UNIQUE NOT NULL,           -- уникальный идентификатор заказа (из JSON)
			track_number VARCHAR(255),                        -- трек-номер для отслеживания
			entry VARCHAR(50),                                -- код входа (например, WBIL)
			locale VARCHAR(10),                               -- локаль (язык) заказа
			internal_signature VARCHAR(255),                  -- внутренняя подпись
			customer_id VARCHAR(255),                         -- идентификатор клиента
			delivery_service VARCHAR(100),                    -- служба доставки
			shardkey VARCHAR(10),                             -- ключ шардирования
			sm_id SMALLINT,                                   -- идентификатор магазина
			date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- дата создания заказа
			oof_shard VARCHAR(10)                             -- шард для OOF (out of stock)
		);
		
		-- создаем индексы для оптимизации запросов
		CREATE INDEX IF NOT EXISTS idx_orders_order_uid ON orders(order_uid);
		CREATE INDEX IF NOT EXISTS idx_orders_track_number ON orders(track_number);
		CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created);
		CREATE INDEX IF NOT EXISTS idx_orders_deleted_at ON orders(deleted_at);
	`

	return db.Exec(sql).Error
}

// createDeliveriesTable создает таблицу deliveries
// каждая запись доставки связана с конкретным заказом через внешний ключ
func createDeliveriesTable(db *gorm.DB) error {
	sql := `
        -- создаем таблицу данных доставки
		CREATE TABLE IF NOT EXISTS deliveries (
			id SERIAL PRIMARY KEY,                                    -- автоинкрементный идентификатор
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,           -- метка времени создания (gorm.Model)
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,           -- метка времени обновления (gorm.Model)
			deleted_at TIMESTAMP NULL,                                -- мягкое удаление (gorm.Model)
			order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE, -- связь с заказом
			name VARCHAR(100) NOT NULL,                               -- имя получателя
			phone VARCHAR(20) NOT NULL,                               -- телефон получателя
			zip VARCHAR(20) NOT NULL,                                 -- почтовый индекс
			city VARCHAR(100) NOT NULL,                               -- город доставки
			address VARCHAR(255) NOT NULL,                            -- адрес доставки
			region VARCHAR(100) NOT NULL,                             -- регион
			email VARCHAR(100) NOT NULL                               -- email получателя
		);
		
		-- создаем индексы для ускорения JOIN-запросов
		CREATE INDEX IF NOT EXISTS idx_deliveries_order_id ON deliveries(order_id);
		CREATE INDEX IF NOT EXISTS idx_deliveries_email ON deliveries(email);
		CREATE INDEX IF NOT EXISTS idx_deliveries_deleted_at ON deliveries(deleted_at);
	`

	return db.Exec(sql).Error
}

// createPaymentsTable создает таблицу payments
// содержит данные о транзакциях и финансовой информации заказа
func createPaymentsTable(db *gorm.DB) error {
	sql := `
		-- создаем таблицу платежной информации
		CREATE TABLE IF NOT EXISTS payments (
			id SERIAL PRIMARY KEY,                                    -- автоинкрементный идентификатор
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,           -- метка времени создания (gorm.Model)
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,           -- метка времени обновления (gorm.Model)
			deleted_at TIMESTAMP NULL,                                -- мягкое удаление (gorm.Model)
			order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE, -- связь с заказом
			transaction VARCHAR(255) UNIQUE NOT NULL,                 -- уникальный идентификатор транзакции
			request_id VARCHAR(255),                                  -- идентификатор запроса платежа
			currency VARCHAR(3) NOT NULL,                             -- валюта платежа (USD, RUB и т.д.)
			provider VARCHAR(50) NOT NULL,                            -- провайдер платежной системы
			amount DECIMAL(10,2) NOT NULL,                            -- общая сумма платежа
			payment_dt BIGINT NOT NULL,                               -- дата платежа в Unix timestamp
			bank VARCHAR(50) NOT NULL,                                -- банк-получатель
			delivery_cost DECIMAL(10,2) DEFAULT 0,                    -- стоимость доставки
			goods_total DECIMAL(10,2) DEFAULT 0,                      -- общая стоимость товаров
			custom_fee DECIMAL(10,2) DEFAULT 0                        -- пользовательская комиссия
		);
		
		-- создаем индексы для быстрого поиска по заказам и транзакциям
		CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_transaction ON payments(transaction);
		CREATE INDEX IF NOT EXISTS idx_payments_deleted_at ON payments(deleted_at);
	`

	return db.Exec(sql).Error
}

// createItemsTable создает таблицу items
// поддерживает множественные товары для одного заказа (one-to-many)
func createItemsTable(db *gorm.DB) error {
	sql := `
        -- создаем таблицу товаров в заказе
        CREATE TABLE IF NOT EXISTS items (
            id SERIAL PRIMARY KEY,                                    -- автоинкрементный идентификатор
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,           -- метка времени создания (gorm.Model)
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,           -- метка времени обновления (gorm.Model)
            deleted_at TIMESTAMP NULL,                                -- мягкое удаление (gorm.Model)
            order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE, -- связь с заказом
            chrt_id INTEGER NOT NULL,                                 -- идентификатор товара в системе
            track_number VARCHAR(255),                                -- трек-номер товара
            price DECIMAL(10,2) NOT NULL,                             -- цена товара
            r_id VARCHAR(100) NOT NULL,                               -- идентификатор записи (GORM преобразует RID в r_id)
            name VARCHAR(255) NOT NULL,                               -- название товара
            sale DECIMAL(5,2) DEFAULT 0,                              -- размер скидки в процентах
            size VARCHAR(20) NOT NULL,                                -- размер товара
            total_price DECIMAL(10,2) NOT NULL,                       -- общая цена с учетом скидки
            nm_id INTEGER NOT NULL,                                   -- идентификатор в marketplace
            brand VARCHAR(100) NOT NULL,                              -- бренд товара
            status SMALLINT NOT NULL                                  -- статус товара
        );
        
        -- создаем индексы для оптимизации запросов по товарам
        CREATE INDEX IF NOT EXISTS idx_items_order_id ON items(order_id);
        CREATE INDEX IF NOT EXISTS idx_items_chrt_id ON items(chrt_id);
        CREATE INDEX IF NOT EXISTS idx_items_nm_id ON items(nm_id);
        CREATE INDEX IF NOT EXISTS idx_items_deleted_at ON items(deleted_at);
	`
	return db.Exec(sql).Error
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
