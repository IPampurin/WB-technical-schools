package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"
	"github.com/jackc/pgx/v5"
)

// CreateItem создаёт новый товар
func (d *DataBase) CreateItem(ctx context.Context, item *domain.Item) error {

	userID := getUserIDFromContext(ctx)

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции CreateItem: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := setUserIDInTransaction(ctx, tx, userID); err != nil {
		return fmt.Errorf("ошибка установки пользователя CreateItem: %w", err)
	}

	dbItem := domainItemToDBItem(item)

	query := `   INSERT INTO items (name, quantity, price, created_at, updated_at) 
	             VALUES ($1, $2, $3, NOW(), NOW())
			  RETURNING id, created_at, updated_at`

	err = tx.QueryRow(ctx, query, dbItem.Name, dbItem.Quantity, dbItem.Price).
		Scan(&dbItem.ID, &dbItem.CreatedAt, &dbItem.UpdatedAt)
	if err != nil {
		return fmt.Errorf("ошибка вставки CreateItem: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита CreateItem: %w", err)
	}

	// заполняем доменную модель полученными из БД полями
	item.ID = dbItem.ID
	item.CreatedAt = dbItem.CreatedAt
	item.UpdatedAt = dbItem.UpdatedAt

	return nil
}

// GetItemByID возвращает товар по его идентификатору
func (d *DataBase) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {

	dbItem := &Item{}

	query := `SELECT id, name, quantity, price, created_at, updated_at
	            FROM items
			   WHERE id = $1`

	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&dbItem.ID, &dbItem.Name, &dbItem.Quantity, &dbItem.Price, &dbItem.CreatedAt, &dbItem.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetItemByID: %w", err)
	}

	return dbItemToDomainItem(dbItem), nil
}

// GetAllItems возвращает список всех товаров
func (d *DataBase) GetAllItems(ctx context.Context) ([]*domain.Item, error) {

	query := `SELECT id, name, quantity, price, created_at, updated_at
	            FROM items
			   ORDER BY id`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса GetAllItems: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.Item, 0)

	for rows.Next() {
		dbItem := &Item{}
		err := rows.Scan(&dbItem.ID, &dbItem.Name, &dbItem.Quantity, &dbItem.Price, &dbItem.CreatedAt, &dbItem.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования GetAllItems: %w", err)
		}
		items = append(items, dbItemToDomainItem(dbItem))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации GetAllItems: %w", err)
	}

	return items, nil
}

// UpdateItem обновляет существующий товар
func (d *DataBase) UpdateItem(ctx context.Context, item *domain.Item) error {

	userID := getUserIDFromContext(ctx)

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции UpdateItem: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := setUserIDInTransaction(ctx, tx, userID); err != nil {
		return fmt.Errorf("ошибка установки пользователя UpdateItem: %w", err)
	}

	dbItem := domainItemToDBItem(item)

	query := `   UPDATE items
	                SET name = $1,
				        quantity = $2,
					    price = $3,
				        updated_at = NOW() 
	              WHERE id = $4
			  RETURNING updated_at`

	err = tx.QueryRow(ctx, query, dbItem.Name, dbItem.Quantity, dbItem.Price, dbItem.ID).
		Scan(&dbItem.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("UpdateItem: товар не найден")
		}
		return fmt.Errorf("ошибка обновления UpdateItem: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита UpdateItem: %w", err)
	}

	item.UpdatedAt = dbItem.UpdatedAt

	return nil
}

// DeleteItem удаляет товар по id
func (d *DataBase) DeleteItem(ctx context.Context, id int) error {

	userID := getUserIDFromContext(ctx)

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции DeleteItem: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := setUserIDInTransaction(ctx, tx, userID); err != nil {
		return fmt.Errorf("ошибка установки пользователя DeleteItem: %w", err)
	}

	query := `DELETE
	            FROM items
			   WHERE id = $1`

	cmdTag, err := tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления DeleteItem: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("DeleteItem: товар не найден")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита DeleteItem: %w", err)
	}

	return nil
}
