package db

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/SalesTracker/pkg/domain"
)

func dbAgregationToDomainAgregation(ag *CategoryAgregation) *domain.CategoryAgregation {

	return &domain.CategoryAgregation{
		Category: ag.Category,
		Sum:      ag.Sum,
		Count:    ag.Count,
	}
}

// GetAnalytics возвращает общие метрики за указанный период
// (параметры from и to должны быть валидными (не zero time))
func (d *DataBase) GetAnalytics(ctx context.Context, from, to time.Time) (sum, avg float64, count int, median, percentile90 float64, err error) {

	// проверяем, что период задан
	if from.IsZero() || to.IsZero() {
		return 0, 0, 0, 0, 0, fmt.Errorf("параметры from и to должны быть указаны")
	}

	query := `SELECT COALESCE(SUM(amount), 0) AS sum,
                     COALESCE(AVG(amount), 0) AS avg,
                     COUNT(*) AS count,
                     COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount), 0) AS median,
                     COALESCE(PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY amount), 0) AS percentile90
                FROM items
               WHERE date BETWEEN $1 AND $2`

	err = d.Pool.QueryRow(ctx, query, from, to).
		Scan(&sum, &avg, &count, &median, &percentile90)
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("ошибка GetAnalytics получения общих метрик: %w", err)
	}

	return
}

// GetAnalyticsByCategory возвращает агрегированные данные по категориям за период
func (d *DataBase) GetAnalyticsByCategory(ctx context.Context, from, to time.Time) ([]*domain.CategoryAgregation, error) {

	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("параметры from и to должны быть указаны")
	}

	query := `SELECT category,
                     COALESCE(SUM(amount), 0) AS sum,
                     COUNT(*) AS count
                FROM items
               WHERE date BETWEEN $1 AND $2
               GROUP BY category
               ORDER BY category`

	rows, err := d.Pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("ошибка GetAnalyticsByCategory при выполнении запроса: %w", err)
	}
	defer rows.Close()

	agregations := make([]*CategoryAgregation, 0)
	for rows.Next() {
		ag := &CategoryAgregation{}
		err := rows.Scan(&ag.Category, &ag.Sum, &ag.Count)
		if err != nil {
			return nil, fmt.Errorf("ошибка GetAnalyticsByCategory при сканировании строки: %w", err)
		}
		agregations = append(agregations, ag)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка GetAnalyticsByCategory при итерации по результатам: %w", err)
	}

	result := make([]*domain.CategoryAgregation, len(agregations))
	for i := range agregations {
		result[i] = dbAgregationToDomainAgregation(agregations[i])
	}

	return result, nil
}
