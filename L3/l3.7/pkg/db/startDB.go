package db

import (
	"context"
	"fmt"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/configuration"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/logger"
)

// DataBase хранит подключение к БД
type DataBase struct {
	*pgxdriver.Postgres
}

// InitDB инициализирует подключение к PostgreSQL и применяет миграции
func InitDB(ctx context.Context, cfgDb *configuration.ConfDB, log logger.Logger) (*DataBase, error) {

	// формируем DSN из конфигурации
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfgDb.User, cfgDb.Password, cfgDb.HostName, cfgDb.Port, cfgDb.Name)

	// создаём клиент pgxdriver с параметрами по умолчанию
	pgxConn, err := pgxdriver.New(dsn, log)
	if err != nil {
		return nil, fmt.Errorf("ошибка InitDB создания клиента pgxdriver: %w", err)
	}
	// defer pg.Close() // закрываем из main()

	// проверяем соединение
	if err = pgxConn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ошибка InitDB соединения с клиентом pgxdriver: %w", err)
	}

	storageDB := &DataBase{pgxConn}

	log.Info("Клиент БД получен.")

	// запускаем миграции
	if err = storageDB.Migration(ctx); err != nil {
		return nil, fmt.Errorf("ошибка миграций: %w", err)
	}

	log.Info("База данных успешно запущена, миграции применены.")

	return storageDB, nil
}

// CloseDB закрывает пул соединений с БД
func CloseDB(storageDB *DataBase) error {

	if storageDB != nil {
		storageDB.Close()
		storageDB = nil
	}

	return nil
}
