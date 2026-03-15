package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/IPampurin/SalesTracker/pkg/domain"
)

// dbItemToDomainItem приводит тип БД к доменному типу
func dbItemToDomainItem(i *Item) *domain.Item {

	return &domain.Item{
		ID:       i.ID,
		Category: i.Category,
		Amount:   i.Amount,
		Date:     i.Date,
	}
}

// GetItems возвращает список записей за указанный период с сортировкой
// (если from или to равны zero time, фильтр по дате не применяется)
// (sortBy может быть "date", "category", "amount"; sortOrder — "asc" или "desc")
func (d *DataBase) GetItems(ctx context.Context, from, to time.Time, sortBy, sortOrder string) ([]*domain.Item, error) {

	// базовый запрос
	query := `SELECT id, category, amount, date
	            FROM items`

	args := make([]interface{}, 0)
	argIdx := 1

	// добавляем фильтр по дате, если задан период
	if !from.IsZero() && !to.IsZero() {
		query += fmt.Sprintf(` WHERE date BETWEEN $%d AND $%d`, argIdx, argIdx+1)
		args = append(args, from, to)
		argIdx += 2
	} else if !from.IsZero() {
		query += fmt.Sprintf(` WHERE date >= $%d`, argIdx)
		args = append(args, from)
		argIdx++
	} else if !to.IsZero() {
		query += fmt.Sprintf(` WHERE date <= $%d`, argIdx)
		args = append(args, to)
		argIdx++
	}

	// добавляем сортировку
	validSortFields := map[string]bool{
		"date":     true,
		"category": true,
		"amount":   true,
	}
	if !validSortFields[sortBy] {
		sortBy = "date" // значение по умолчанию
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc" // по умолчанию новые сверху
	}
	query += fmt.Sprintf(` ORDER BY %s %s`, sortBy, sortOrder)

	// выполняем запрос
	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка GetItems при выполнении запроса: %w", err)
	}
	defer rows.Close()

	items := make([]*Item, 0)
	for rows.Next() {
		i := &Item{}
		err := rows.Scan(
			&i.ID,
			&i.Category,
			&i.Amount,
			&i.Date)
		if err != nil {
			return nil, fmt.Errorf("ошибка GetItems при сканировании строки: %w", err)
		}

		items = append(items, i)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка GetItems при итерации по результатам: %w", err)
	}

	// преобразуем в доменные объекты
	result := make([]*domain.Item, len(items))
	for i := range items {
		result[i] = dbItemToDomainItem(items[i])
	}

	return result, nil
}

// CreateItem добавляет новую запись в БД и возвращает её с заполненным ID
func (d *DataBase) CreateItem(ctx context.Context, item *domain.Item) (*domain.Item, error) {

	query := `   INSERT INTO items(category, amount, date)
	             VALUES ($1, $2, $3)
			  RETURNING id, category, amount, date`

	i := &Item{}

	err := d.Pool.QueryRow(ctx, query, item.Category, item.Amount, item.Date).
		Scan(&i.ID, &i.Category, &i.Amount, &i.Date)
	if err != nil {
		return nil, fmt.Errorf("ошибка CreateItem добавления записи в items: %w", err)
	}

	return dbItemToDomainItem(i), nil
}

// UpdateItem обновляет существующую запись
// (возвращает обновлённую запись или ошибку, если запись с таким ID не найдена)
func (d *DataBase) UpdateItem(ctx context.Context, id int, item *domain.Item) (*domain.Item, error) {

	query := `   UPDATE items
	                SET category = $1, amount = $2, date = $3
				  WHERE id = $4
			  RETURNING id, category, amount, date`

	i := &Item{}

	err := d.Pool.QueryRow(ctx, query, item.Category, item.Amount, item.Date, id).
		Scan(&i.ID, &i.Category, &i.Amount, &i.Date)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("ошибка UpdateItem обновления записи в items: записи с id = %d не существует", id)
		}
		return nil, fmt.Errorf("ошибка UpdateItem обновления записи в items: %w", err)
	}

	return dbItemToDomainItem(i), nil
}

// DeleteItem удаляет запись по идентификатору
// (возвращает ошибку, если запись не существует)
func (d *DataBase) DeleteItem(ctx context.Context, id int) error {

	query := `DELETE
	            FROM items
			   WHERE id = $1`

	cmd, err := d.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка DeleteItem удаления записи из items: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("ошибка DeleteItem удаления записи в items: записи с id = %d не существует", id)
	}

	return nil
}
