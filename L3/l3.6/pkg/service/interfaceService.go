package service

import (
	"context"
	"time"

	"github.com/IPampurin/SalesTracker/pkg/domain"
)

// ServiceMethods определяет методы бизнес-логики
type ServiceMethods interface {

	// GetItems возвращает список записей за указанный период с сортировкой
	// (если from или to равны zero time, фильтр по дате не применяется)
	// (sortBy может быть "date", "category", "amount"; sortOrder — "asc" или "desc")
	GetItems(ctx context.Context, from, to time.Time, sortBy, sortOrder string) ([]*domain.Item, error)

	// CreateItem добавляет новую запись в БД и возвращает её с заполненным ID
	CreateItem(ctx context.Context, item *domain.Item) (*domain.Item, error)

	// UpdateItem обновляет существующую запись
	// (возвращает обновлённую запись или ошибку, если запись с таким ID не найдена)
	UpdateItem(ctx context.Context, id int, item *domain.Item) (*domain.Item, error)

	// DeleteItem удаляет запись по идентификатору
	// (возвращает ошибку, если запись не существует)
	DeleteItem(ctx context.Context, id int) error

	// GetAnalytics возвращает общие метрики за указанный период
	// (параметры from и to должны быть валидными (не zero time))
	GetAnalytics(ctx context.Context, from, to time.Time) (sum, avg float64, count int, median, percentile90 float64, err error)

	// GetAnalyticsByCategory возвращает агрегированные данные по категориям за период
	GetAnalyticsByCategory(ctx context.Context, from, to time.Time) ([]*domain.CategoryAgregation, error)

	// CSVExporter возвращает файл для скачивания
	CSVExporter()
}
