package db

import (
	"context"
	"fmt"
	"net"
	"time"
)

// SaveAnalytics записывает каждый переход
func (d *DataBase) SaveAnalytics(ctx context.Context, linkID int, accessedAt time.Time, userAgent, ipAddress, referer string) error {

	query := `INSERT INTO analytics (link_id, accessed_at, user_agent, ip_address, referer)
              VALUES ($1, $2, $3, $4, $5)`

	ip := net.ParseIP(ipAddress) // если пусто, вернёт nil
	_, err := d.Pool.Exec(ctx, query, linkID, accessedAt, userAgent, ip, referer)
	if err != nil {
		return fmt.Errorf("ошибка добавления записи о переходе в SaveAnalytics: %w", err)
	}

	return nil
}

// GetAnalyticsByLinkID получение всех записей для конкретной ссылки
func (d *DataBase) GetAnalyticsByLinkID(ctx context.Context, linkID int) ([]*Analytics, error) {

	query := `SELECT *
	            FROM analytics
			   WHERE link_id = $1`

	rows, err := d.Pool.Query(ctx, query, linkID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка записей в GetAnalyticsByLinkID: %w", err)
	}
	defer rows.Close()

	var analytics []*Analytics
	for rows.Next() {
		var a Analytics
		err := rows.Scan(
			&a.ID,
			&a.LinkID,
			&a.AccessedAt,
			&a.UserAgent,
			&a.IPAddress,
			&a.Referer,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки списка записей в GetAnalyticsByLinkID: %w", err)
		}

		analytics = append(analytics, &a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по списку записей в GetAnalyticsByLinkID: %w", err)
	}

	return analytics, nil
}

// агрегация

// CountClicksByDay - группировка по дням
func (d *DataBase) CountClicksByDay(ctx context.Context, linkID int, from, to time.Time) (map[string]int, error) {

	query := `SELECT DATE(accessed_at) AS day,
	                 COUNT(*) AS count
                FROM analytics
               WHERE link_id = $1
                 AND accessed_at >= $2 AND accessed_at < $3
               GROUP BY day`

	rows, err := d.Pool.Query(ctx, query, linkID, from, to)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса в CountClicksByDay: %w", err)
	}
	defer rows.Close()

	dayCountClick := make(map[string]int)
	var key string
	var val int
	for rows.Next() {
		err := rows.Scan(
			&key,
			&val,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки запроса в CountClicksByDay: %w", err)
		}

		dayCountClick[key] = val
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по списку записей в CountClicksByDay: %w", err)
	}

	return dayCountClick, nil
}

// CountClicksByMonth - группировка по месяцам
func (d *DataBase) CountClicksByMonth(ctx context.Context, linkID int, from, to time.Time) (map[string]int, error) {

	query := `SELECT TO_CHAR(accessed_at, 'YYYY-MM') AS month,
	                 COUNT(*) AS count
                FROM analytics
               WHERE link_id = $1
                 AND accessed_at >= $2 AND accessed_at < $3
               GROUP BY month`

	rows, err := d.Pool.Query(ctx, query, linkID, from, to)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса в CountClicksByMonth: %w", err)
	}
	defer rows.Close()

	monthCountClick := make(map[string]int)
	var key string
	var val int
	for rows.Next() {
		err := rows.Scan(
			&key,
			&val,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки запроса в CountClicksByMonth: %w", err)
		}

		monthCountClick[key] = val
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по списку записей в CountClicksByMonth: %w", err)
	}

	return monthCountClick, nil
}

// CountClicksByUserAgent - группировка по User-Agent
func (d *DataBase) CountClicksByUserAgent(ctx context.Context, linkID int) (map[string]int, error) {

	query := `SELECT user_agent,
	                 COUNT(*) AS count
                FROM analytics
               WHERE link_id = $1
			   GROUP BY user_agent`

	rows, err := d.Pool.Query(ctx, query, linkID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса в CountClicksByUserAgent: %w", err)
	}
	defer rows.Close()

	userAgentCountClick := make(map[string]int)
	var key string
	var val int
	for rows.Next() {
		err := rows.Scan(
			&key,
			&val,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки запроса в CountClicksByUserAgent: %w", err)
		}

		userAgentCountClick[key] = val
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по списку записей в CountClicksByUserAgent: %w", err)
	}

	return userAgentCountClick, nil
}
