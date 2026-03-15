package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/IPampurin/SalesTracker/pkg/domain"
)

// GetItems возвращает список записей за указанный период с сортировкой
// (если from или to равны zero time, фильтр по дате не применяется)
// (sortBy может быть "date", "category", "amount"; sortOrder — "asc" или "desc")
func (s *Service) GetItems(ctx context.Context, from, to time.Time, sortBy, sortOrder string) ([]*domain.Item, error) {

	return s.storage.GetItems(ctx, from, to, sortBy, sortOrder)
}

// CreateItem добавляет новую запись в БД и возвращает её с заполненным ID
func (s *Service) CreateItem(ctx context.Context, item *domain.Item) (*domain.Item, error) {

	// проверяем категории
	if !isValidCategory(item.Category) {
		return nil, fmt.Errorf("недопустимая категория")
	}
	// проверяем, что дата не в будущем
	if item.Date.After(time.Now()) {
		return nil, fmt.Errorf("дата не может быть в будущем")
	}

	return s.storage.CreateItem(ctx, item)
}

// UpdateItem обновляет существующую запись
// (возвращает обновлённую запись или ошибку, если запись с таким ID не найдена)
func (s *Service) UpdateItem(ctx context.Context, id int, item *domain.Item) (*domain.Item, error) {

	// проверяем категории
	if !isValidCategory(item.Category) {
		return nil, fmt.Errorf("недопустимая категория")
	}
	// проверяем, что дата не в будущем
	if item.Date.After(time.Now()) {
		return nil, fmt.Errorf("дата не может быть в будущем")
	}

	return s.storage.UpdateItem(ctx, id, item)
}

// DeleteItem удаляет запись по идентификатору
// (возвращает ошибку, если запись не существует)
func (s *Service) DeleteItem(ctx context.Context, id int) error {
	return s.storage.DeleteItem(ctx, id)
}

// GetAnalytics возвращает общие метрики за указанный период
// (параметры from и to должны быть валидными (не zero time))
func (s *Service) GetAnalytics(ctx context.Context, from, to time.Time) (sum, avg float64, count int, median, percentile90 float64, err error) {
	return s.storage.GetAnalytics(ctx, from, to)
}

// GetAnalyticsByCategory возвращает агрегированные данные по категориям за период
func (s *Service) GetAnalyticsByCategory(ctx context.Context, from, to time.Time) ([]*domain.CategoryAgregation, error) {
	return s.storage.GetAnalyticsByCategory(ctx, from, to)
}

// ExportCSV возвращает данные за период в формате CSV как []byte
func (s *Service) ExportCSV(ctx context.Context, from, to time.Time) ([]byte, error) {

	items, err := s.storage.GetItems(ctx, from, to, "date", "asc")
	if err != nil {
		return nil, err
	}

	// генерация CSV в памяти
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// заголовки
	if err := writer.Write([]string{"ID", "Дата", "Категория", "Сумма"}); err != nil {
		return nil, err
	}

	for _, item := range items {
		record := []string{
			strconv.Itoa(item.ID),
			item.Date.Format("2006-01-02"),
			item.Category,
			strconv.FormatFloat(item.Amount, 'f', 2, 64),
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}

	// записываем буфер
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// isValidCategory проверяет, что категория входит в допустимый список.
func isValidCategory(category string) bool {

	switch category {
	case domain.ProductCategory,
		domain.TransportCategory,
		domain.EntertainmentCategory,
		domain.HealthCategory,
		domain.OtherCategory:
		return true
	default:
		return false
	}
}
