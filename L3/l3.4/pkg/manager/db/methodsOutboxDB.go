package db

import (
	"context"
	"fmt"

	"github.com/IPampurin/ImageProcessor/pkg/domain"
	"github.com/google/uuid"
)

// domOutboxToDbOutbox преобразует доменную модель OutboxData в модель Outbox локальной БД
func domOutboxToDbOutbox(domOut *domain.OutboxData) *Outbox {

	return &Outbox{
		ID:        domOut.ID,
		Topic:     domOut.Topic,
		Key:       domOut.Key,
		Payload:   domOut.Payload,
		CreatedAt: domOut.CreatedAt,
	}
}

// dbOutboxToDomOutboxData преобразует модель Outbox локальной БД в доменную модель OutboxData
func dbOutboxToDomOutboxData(dbOut *Outbox) *domain.OutboxData {

	return &domain.OutboxData{
		ID:        dbOut.ID,
		Topic:     dbOut.Topic,
		Key:       dbOut.Key,
		Payload:   dbOut.Payload,
		CreatedAt: dbOut.CreatedAt,
	}
}

// CreateOutbox создаёт запись в таблице outbox
func (d *DataBase) CreateOutbox(ctx context.Context, dataToSend *domain.OutboxData) error {

	rowToSend := domOutboxToDbOutbox(dataToSend)

	query := `INSERT INTO outbox (id, topic, key, payload, created_at)
	          VALUES ($1, $2, $3, $4, $5)`

	_, err := d.Pool.Exec(ctx, query,
		rowToSend.ID,
		rowToSend.Topic,
		rowToSend.Key,
		rowToSend.Payload,
		rowToSend.CreatedAt)
	if err != nil {
		return fmt.Errorf("ошибка CreateOutbox при добавлении записи в outbox: %w", err)
	}

	return nil
}

// GetUnsentOutbox получает limit записей для отправки брокеру
func (d *DataBase) GetUnsentOutbox(ctx context.Context, limit int) ([]*domain.OutboxData, error) {

	query := `SELECT *
	            FROM outbox
			   ORDER BY created_at
			   LIMIT $1`

	rows, err := d.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка GetUnsentOutbox при получении записей из outbox: %w", err)
	}
	defer rows.Close()

	rowsToSend := make([]*Outbox, 0)
	for rows.Next() {
		outRow := &Outbox{}
		err := rows.Scan(
			&outRow.ID,
			&outRow.Topic,
			&outRow.Key,
			&outRow.Payload,
			&outRow.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка GetUnsentOutbox при сканировании записи из outbox: %w", err)
		}

		rowsToSend = append(rowsToSend, outRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка GetUnsentOutbox при итерации по записям из outbox: %w", err)
	}

	result := make([]*domain.OutboxData, len(rowsToSend))
	for i := range rowsToSend {
		result[i] = dbOutboxToDomOutboxData(rowsToSend[i])
	}

	return result, nil
}

// DeleteOutbox удаляет запись из таблицы outbox
func (d *DataBase) DeleteOutbox(ctx context.Context, uid uuid.UUID) error {

	query := `DELETE
	            FROM outbox
			   WHERE id = $1`

	_, err := d.Pool.Exec(ctx, query, uid)
	if err != nil {
		return fmt.Errorf("ошибка DeleteOutbox при удалении записи из outbox: %w", err)
	}

	return nil
}
