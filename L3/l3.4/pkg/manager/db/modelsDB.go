package db

import (
	"time"

	"github.com/google/uuid"
)

// Image представляет запись в таблице images,
// хранящей как исходные изображения, так и все их обработанные варианты.
type Image struct {
	ID           uuid.UUID  `db:"id"`            // уникальный идентификатор файла
	OriginalID   *uuid.UUID `db:"original_id"`   // ссылка на исходный файл (NULL для исходного)
	Name         string     `db:"name"`          // полное имя файла (с суффиксом)
	Type         string     `db:"type"`          // тип: 'original', 'resized', 'thumbnail', 'watermarked'
	ContentType  string     `db:"content_type"`  // MIME-тип (image/jpeg, image/png)
	Size         int64      `db:"size"`          // размер файла в байтах
	Width        *int       `db:"width"`         // ширина (если есть)
	Height       *int       `db:"height"`        // высота (если есть)
	Status       string     `db:"status"`        // статус обработки (для исходного: pending/processing/completed/failed; для вариантов: completed)
	ErrorMessage *string    `db:"error_message"` // текст ошибки (только для исходного при статусе failed)
	StoragePath  string     `db:"storage_path"`  // ключ в S3
	CreatedAt    time.Time  `db:"created_at"`    // время загрузки с фронта
}

// Outbox представляет запись в таблице outbox
// (сообщение для отправки в Kafka (Transactional Outbox pattern))
type Outbox struct {
	ID        uuid.UUID `db:"id"`         // уникальный идентификатор сообщения
	Topic     string    `db:"topic"`      // топик Kafka ('image-tasks')
	Key       string    `db:"key"`        // ключ сообщения (image_id)
	Payload   []byte    `db:"payload"`    // данные задачи (сериализованный Task)
	CreatedAt time.Time `db:"created_at"` // время создания записи
}
