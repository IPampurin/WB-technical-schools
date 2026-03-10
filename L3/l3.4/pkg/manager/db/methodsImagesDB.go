package db

import (
	"context"
	"fmt"

	"github.com/IPampurin/ImageProcessor/pkg/domain"
	"github.com/google/uuid"
)

// domImageToDbImage преобразует доменную модель ImageData в модель Image локальной БД
func domImageToDbImage(iData *domain.ImageData) *Image {

	return &Image{
		ID:           iData.ID,
		OriginalID:   iData.OriginalID,
		Name:         iData.Name,
		Type:         iData.Type,
		ContentType:  iData.ContentType,
		Size:         iData.Size,
		Width:        iData.Width,
		Height:       iData.Height,
		Status:       iData.Status,
		ErrorMessage: iData.ErrorMessage,
		StoragePath:  iData.StoragePath,
		CreatedAt:    iData.CreatedAt,
	}
}

// dbImageToDomImage преобразует модель Image локальной БД в доменную модель ImageData
func dbImageToDomImage(i *Image) *domain.ImageData {

	return &domain.ImageData{
		ID:           i.ID,
		OriginalID:   i.OriginalID,
		Name:         i.Name,
		Type:         i.Type,
		ContentType:  i.ContentType,
		Size:         i.Size,
		Width:        i.Width,
		Height:       i.Height,
		Status:       i.Status,
		ErrorMessage: i.ErrorMessage,
		StoragePath:  i.StoragePath,
		CreatedAt:    i.CreatedAt,
	}
}

// InsertImage создаёт запись в images
func (d *DataBase) InsertImage(ctx context.Context, iData *domain.ImageData) error {

	i := domImageToDbImage(iData)

	query := `INSERT INTO images (id, original_id, name, type, content_type, size, width, height, status, error_message, storage_path, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := d.Pool.Exec(ctx, query,
		i.ID,
		i.OriginalID,
		i.Name,
		i.Type,
		i.ContentType,
		i.Size,
		i.Width,
		i.Height,
		i.Status,
		i.ErrorMessage,
		i.StoragePath,
		i.CreatedAt)
	if err != nil {
		return fmt.Errorf("ошибка InsertImage добавления записи в images: %w", err)
	}

	return nil
}

// GetByID возвращает запись из images по уникальному идентификатору
func (d *DataBase) GetByID(ctx context.Context, uid uuid.UUID) (*domain.ImageData, error) {

	query := `SELECT *
	            FROM images
			   WHERE id = $1`

	i := &Image{}

	err := d.Pool.QueryRow(ctx, query, uid).Scan(
		&i.ID,
		&i.OriginalID,
		&i.Name,
		&i.Type,
		&i.ContentType,
		&i.Size,
		&i.Width,
		&i.Height,
		&i.Status,
		&i.ErrorMessage,
		&i.StoragePath,
		&i.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("ошибка GetByID получения записи в images: %w", err)
	}

	return dbImageToDomImage(i), nil
}

// UpdateStatusOrErr обновляет запсь в images по статусу или ошибке
func (d *DataBase) UpdateStatusOrErr(ctx context.Context, uid uuid.UUID, status string, errMsg *string) error {

	query := `UPDATE images
	             SET status = $1, error_message = $2
			   WHERE id = $3`

	_, err := d.Pool.Exec(ctx, query, status, errMsg, uid)
	if err != nil {
		return fmt.Errorf("ошибка UpdateStatusOrErr при изменении записи в images: %w", err)
	}

	return nil
}

// DeleteImage удаляет запись из images
func (d *DataBase) DeleteImage(ctx context.Context, uid uuid.UUID) error {

	query := `DELETE
	            FROM images
	           WHERE id = $1`

	_, err := d.Pool.Exec(ctx, query, uid)
	if err != nil {
		return fmt.Errorf("ошибка DeleteImage при удалении записи из images: %w", err)
	}

	return nil
}

// ListLatestOriginals используется для отображения UI изображений в галерее
func (d *DataBase) ListLatestOriginals(ctx context.Context, limit int) ([]*domain.ImageData, error) {

	query := `SELECT *
	            FROM images
			   WHERE original_id IS NULL
			   ORDER BY created_at DESC
			   LIMIT $1`

	rows, err := d.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка ListLatestOriginals при получении записей из images: %w", err)
	}
	defer rows.Close()

	images := make([]*Image, 0)
	for rows.Next() {
		i := &Image{}
		err := rows.Scan(
			&i.ID,
			&i.OriginalID,
			&i.Name,
			&i.Type,
			&i.ContentType,
			&i.Size,
			&i.Width,
			&i.Height,
			&i.Status,
			&i.ErrorMessage,
			&i.StoragePath,
			&i.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка ListLatestOriginals при сканировании записи из images: %w", err)
		}

		images = append(images, i)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка ListLatestOriginals при итерации по записям из images: %w", err)
	}

	result := make([]*domain.ImageData, len(images))
	for i := range result {
		result[i] = dbImageToDomImage(images[i])
	}

	return result, nil
}

// GetVariantsByOriginalID возвращает все варианты для указанного оригинала
func (d *DataBase) GetVariantsByOriginalID(ctx context.Context, originalID uuid.UUID) ([]*domain.ImageData, error) {

	query := `SELECT *
	            FROM images
			   WHERE original_id = $1`

	rows, err := d.Pool.Query(ctx, query, originalID)
	if err != nil {
		return nil, fmt.Errorf("ошибка GetVariantsByOriginalID: %w", err)
	}
	defer rows.Close()

	variants := make([]*domain.ImageData, 0)
	for rows.Next() {
		i := &Image{}
		err := rows.Scan(
			&i.ID, &i.OriginalID, &i.Name, &i.Type, &i.ContentType,
			&i.Size, &i.Width, &i.Height, &i.Status, &i.ErrorMessage,
			&i.StoragePath, &i.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка GetVariantsByOriginalID сканирования варианта: %w", err)
		}

		variants = append(variants, dbImageToDomImage(i))
	}

	return variants, nil
}
