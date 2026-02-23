package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IPampurin/DelayedNotifier/pkg/configuration"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/dbpg"
)

// константы статусов уведомлений
const (
	StatusScheduled  = "scheduled"  // надо отправить
	StatusPublishing = "publishing" // в рэббите или консумере
	StatusSent       = "sent"       // отправлено
	StatusFailed     = "failed"     // ошибка отправки
	StatusCancelled  = "cancelled"  // отменено
)

// Notification — структура, описывающая уведомление и соответствующая строке в таблице notifications БД
type Notification struct {
	UID        uuid.UUID    `json:"uid"`                  // uid, UUID для глобальной уникальности
	UserID     int          `json:"user_id"`              // user_id, адресат уведомления (кому)
	Channel    []string     `json:"channel"`              // channel, канал доставки (email, telegram)
	Content    string       `json:"content"`              // content, само уведомление
	Status     string       `json:"status"`               // status, текущий статус (scheduled/publishing/sent/failed/cancelled)
	SendFor    time.Time    `json:"send_for"`             // send_for, когда планируется отправить (гггг.мм.дд чч:мм:сс)
	SendAt     sql.NullTime `json:"send_at,omitempty"`    // send_at, фактическое время отправки (гггг.мм.дд чч:мм:сс), момент получения Consumer подтверждения от внешнего API
	RetryCount int          `json:"retry_count"`          // retry_count, счетчик попыток отправки (для Consumer)
	LastError  string       `json:"last_error,omitempty"` // last_error, информация о сбое при крайней отправке
	CreatedAt  time.Time    `json:"created_at"`           // created_at, время создания
}

// ClientPostgres хранит подключение к БД
// делаем его публичным, чтобы другие пакеты могли использовать методы
type ClientPostgres struct {
	*dbpg.DB
}

// глобальный экземпляр клиента (синглтон)
var defaultClient *ClientPostgres

// InitDB инициализирует подключение к PostgreSQL
func InitDB(cfg *configuration.ConfDB) error {

	// формируем DSN для master (и slaves, если потребуется)
	// формат: postgres://user:pass@host:port/dbname?sslmode=disable
	masterDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.HostName, cfg.Port, cfg.Name)

	// опции подключения (можно вынести и в конфиг)
	opts := &dbpg.Options{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// создаём подключение (пока без слейвов)
	dbConn, err := dbpg.New(masterDSN, nil, opts)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	defaultClient = &ClientPostgres{dbConn}

	// применяем миграции
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := defaultClient.Migration(ctx); err != nil {
		return fmt.Errorf("ошибка миграции: %w", err)
	}

	return nil
}

// CloseDB закрывает соединение с БД
func CloseDB() error {

	if defaultClient != nil && defaultClient.Master != nil {
		return defaultClient.Master.Close()
	}

	return nil
}

// GetClientDB возвращает глобальный экземпляр клиента БД
func GetClientDB() *ClientPostgres {

	return defaultClient
}

// CreateNotification создаёт новое уведомление в БД
func (c *ClientPostgres) CreateNotification(ctx context.Context, n *Notification) error {

	query := `INSERT INTO notifications (uid, user_id, channel, content, status, send_for, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := c.ExecContext(ctx, query, n.UID, n.UserID, dbpg.Array(&n.Channel), n.Content, n.Status, n.SendFor, n.CreatedAt)
	if err != nil {
		return fmt.Errorf("ошибка запроса CreateNotification: %w", err)
	}

	return nil
}

// GetNotification возвращает уведомление по UID
func (c *ClientPostgres) GetNotification(ctx context.Context, uid uuid.UUID) (*Notification, error) {

	query := `SELECT uid, user_id, channel,
	                 content, status, send_for, send_at,
					 retry_count, last_error, created_at
	            FROM notifications
			   WHERE uid = $1`
	var n Notification
	var channelSlice []string
	err := c.QueryRowContext(ctx, query, uid).Scan(
		&n.UID, &n.UserID, dbpg.Array(&channelSlice),
		&n.Content, &n.Status, &n.SendFor, &n.SendAt,
		&n.RetryCount, &n.LastError, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса GetNotification: %w", err)
	}
	n.Channel = channelSlice

	return &n, nil
}

// CancelNotification помечает уведомление как отменённое (статус cancelled)
// возвращает sql.ErrNoRows, если уведомление с таким uid не найдено или уже не в статусе 'scheduled'
func (c *ClientPostgres) CancelNotification(ctx context.Context, uid uuid.UUID) error {

	query := `UPDATE notifications
	             SET status = 'cancelled'
			   WHERE uid = $1 AND status = $2`
	result, err := c.ExecContext(ctx, query, uid, StatusScheduled)
	if err != nil {
		return fmt.Errorf("ошибка запроса CancelNotification: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows // ошибка "уведомление не найдено или уже не scheduled"
	}

	return nil
}

// UpdateNotificationStatus обновляет статус и связанные поля (для consumer)
// sentAt может быть nil, если отправка ещё не производилась
func (c *ClientPostgres) UpdateNotificationStatus(ctx context.Context, uid uuid.UUID, status string, sentAt *time.Time, retryCount int, lastError string) error {

	query := `UPDATE notifications
                 SET status = $1, send_at = $2, retry_count = $3, last_error = $4
		       WHERE uid = $5`
	_, err := c.ExecContext(ctx, query, status, sentAt, retryCount, lastError, uid)
	if err != nil {
		return fmt.Errorf("ошибка запроса UpdateNotificationStatus: %w", err)
	}

	return nil
}

// GetNotification возвращает уведомления созданные позднее указанной даты (для прогрева кэша)
func (c *ClientPostgres) GetNotificationsLastPeriod(ctx context.Context, lastPeriod time.Duration) ([]*Notification, error) {

	intervalStr := fmt.Sprintf("%d seconds", int(lastPeriod.Seconds()))
	query := `SELECT uid, user_id, channel,
	                 content, status, send_for, send_at,
					 retry_count, last_error, created_at
	            FROM notifications
			   WHERE created_at >= (NOW() - $1::interval)`

	rows, err := c.QueryContext(ctx, query, intervalStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса GetNotificationsLastPeriod: %w", err)
	}
	defer rows.Close()

	notifications := make([]*Notification, 0)

	for rows.Next() {
		var n Notification
		var channelSlice []string
		err := rows.Scan(
			&n.UID, &n.UserID, dbpg.Array(&channelSlice),
			&n.Content, &n.Status, &n.SendFor, &n.SendAt,
			&n.RetryCount, &n.LastError, &n.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования GetNotificationsLastPeriod: %w", err)
		}
		n.Channel = channelSlice
		notifications = append(notifications, &n)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации GetNotificationsLastPeriod: %w", err)
	}

	return notifications, nil
}

// GetScheduledNotifications возвращает уведомления для отправки в Rabbit планировщиком
// (статус scheduled, время отправки теоретически +-(ConfScheduler.Interval)/2 текущему моменту)
func (c *ClientPostgres) GetScheduledNotifications(ctx context.Context, interval time.Duration) ([]*Notification, error) {

	intervalStr := fmt.Sprintf("%d seconds", int(interval.Seconds()))
	query := `SELECT uid, user_id, channel,
	                 content, status, send_for, send_at,
					 retry_count, last_error, created_at
	            FROM notifications
			   WHERE status = $1 AND send_for <= (NOW() + $2::interval/2)` // нижнюю границу на всякий случай не указываем

	rows, err := c.QueryContext(ctx, query, StatusScheduled, intervalStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса GetScheduledNotifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var n Notification
		var channelSlice []string
		err := rows.Scan(
			&n.UID, &n.UserID, dbpg.Array(&channelSlice),
			&n.Content, &n.Status, &n.SendFor, &n.SendAt,
			&n.RetryCount, &n.LastError, &n.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования GetScheduledNotifications: %w", err)
		}
		n.Channel = channelSlice
		notifications = append(notifications, &n)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации GetScheduledNotifications: %w", err)
	}

	return notifications, nil
}
