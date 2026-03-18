package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"
	"github.com/jackc/pgx/v5"
)

// GetHistory возвращает записи истории с фильтрацией и пагинацией
func (d *DataBase) GetHistory(ctx context.Context, filter *domain.HistoryFilter) ([]*domain.ItemHistory, error) {

	// преобразуем доменный фильтр в фильтр для БД
	dbFilter := domainHistoryFilterToDBHistoryFilter(filter)

	// базовый запрос
	query := `SELECT id, item_id, action, old_data, new_data, changed_by, changed_at
		        FROM item_history
		       WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	// добавляем условия фильтрации
	if dbFilter.ItemID != 0 {
		query += fmt.Sprintf(" AND item_id = $%d", argPos)
		args = append(args, dbFilter.ItemID)
		argPos++
	}
	if !dbFilter.FromDate.IsZero() {
		query += fmt.Sprintf(" AND changed_at >= $%d", argPos)
		args = append(args, dbFilter.FromDate)
		argPos++
	}
	if !dbFilter.ToDate.IsZero() {
		query += fmt.Sprintf(" AND changed_at <= $%d", argPos)
		args = append(args, dbFilter.ToDate)
		argPos++
	}
	if dbFilter.UserID != 0 {
		query += fmt.Sprintf(" AND changed_by = $%d", argPos)
		args = append(args, dbFilter.UserID)
		argPos++
	}
	if dbFilter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argPos)
		args = append(args, dbFilter.Action)
		argPos++
	}

	// добавляем сортировку (по умолчанию по убыванию даты)
	query += " ORDER BY changed_at DESC"

	// пагинация
	if dbFilter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, dbFilter.Limit)
		argPos++
	}
	if dbFilter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, dbFilter.Offset)
		// argPos++
	}

	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса GetHistory: %w", err)
	}
	defer rows.Close()

	history := make([]*domain.ItemHistory, 0)
	for rows.Next() {

		dbHistory := &ItemHistory{}
		err := rows.Scan(&dbHistory.ID, &dbHistory.ItemID, &dbHistory.Action, &dbHistory.OldData, &dbHistory.NewData, &dbHistory.ChangedBy, &dbHistory.ChangedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования GetHistory: %w", err)
		}

		history = append(history, dbItemHistoryToDomainItemHistory(dbHistory))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации GetHistory: %w", err)
	}

	return history, nil
}

// GetHistoryByID возвращает одну запись истории по её ID
func (d *DataBase) GetHistoryByID(ctx context.Context, id int) (*domain.ItemHistory, error) {

	dbHistory := &ItemHistory{}

	query := `SELECT id, item_id, action, old_data, new_data, changed_by, changed_at 
	            FROM item_history
			   WHERE id = $1`

	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&dbHistory.ID, &dbHistory.ItemID, &dbHistory.Action, &dbHistory.OldData, &dbHistory.NewData, &dbHistory.ChangedBy, &dbHistory.ChangedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetHistoryByID: %w", err)
	}

	return dbItemHistoryToDomainItemHistory(dbHistory), nil
}

// GetHistoryCount возвращает общее количество записей, удовлетворяющих фильтру
func (d *DataBase) GetHistoryCount(ctx context.Context, filter *domain.HistoryFilter) (int, error) {

	dbFilter := domainHistoryFilterToDBHistoryFilter(filter)

	query := `SELECT COUNT(*)
	            FROM item_history
			   WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	if dbFilter.ItemID != 0 {
		query += fmt.Sprintf(" AND item_id = $%d", argPos)
		args = append(args, dbFilter.ItemID)
		argPos++
	}
	if !dbFilter.FromDate.IsZero() {
		query += fmt.Sprintf(" AND changed_at >= $%d", argPos)
		args = append(args, dbFilter.FromDate)
		argPos++
	}
	if !dbFilter.ToDate.IsZero() {
		query += fmt.Sprintf(" AND changed_at <= $%d", argPos)
		args = append(args, dbFilter.ToDate)
		argPos++
	}
	if dbFilter.UserID != 0 {
		query += fmt.Sprintf(" AND changed_by = $%d", argPos)
		args = append(args, dbFilter.UserID)
		argPos++
	}
	if dbFilter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argPos)
		args = append(args, dbFilter.Action)
		// argPos++
	}

	count := 0
	err := d.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetHistoryCount: %w", err)
	}

	return count, nil
}
