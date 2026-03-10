package domain

import (
	"io"
	"time"

	"github.com/google/uuid"
)

// UploadData - структура для service после запроса фронтэнда (POST /upload)
type UploadData struct {
	Filename    string         // оригинальное имя файла
	ContentType string         // MIME-тип (image/jpeg и тп)
	Size        int64          // размер в байтах
	Reader      io.Reader      // поток данных файла
	Thumbnail   bool           // нужна ли миниатюра
	Watermark   bool           // нужен ли водяной знак
	Resize      *ResizeOptions // параметры ресайза (если есть)
}

// ImageData представляет данные о файле для сохранения в БД,
type ImageData struct {
	ID           uuid.UUID  // уникальный идентификатор файла
	OriginalID   *uuid.UUID // ссылка на исходный файл (NULL для исходного)
	Name         string     // полное имя файла (с суффиксом)
	Type         string     // тип: 'original', 'resized', 'thumbnail', 'watermarked'
	ContentType  string     // MIME-тип (image/jpeg, image/png)
	Size         int64      // размер файла в байтах
	Width        *int       // ширина (если есть)
	Height       *int       // высота (если есть)
	Status       string     // статус обработки (для исходного: pending/processing/completed/failed; для вариантов: completed)
	ErrorMessage *string    // текст ошибки (только для исходного при статусе failed)
	StoragePath  string     // ключ в S3
	CreatedAt    time.Time  // время загрузки с фронта
}

// OutboxData представляет данные о задаче для внесения в БД
type OutboxData struct {
	ID        uuid.UUID // уникальный идентификатор сообщения
	Topic     string    // топик Kafka ('image-tasks')
	Key       string    // ключ сообщения (image_id)
	Payload   []byte    // данные задачи (сериализованный Task)
	CreatedAt time.Time // время создания записи
}

// ImageTask аккумулирует данные для загрузки во внешнее S3 хранилище
type ImageTask struct {
	ImageID      string         // UUID файла
	ObjectKey    string         // ключ для S3 хранилища
	Bucket       string         // имя бакета в S3 хранилище
	Thumbnail    bool           // требуется ли миниатюра
	Watermark    bool           // нужен ли водяной знак
	Resize       *ResizeOptions // требуемые размеры изображения
	OriginalName string         // имя файла
}

// ResizeOptions описывает размеры изображения при resize
type ResizeOptions struct {
	Width  int // ширина изображения
	Height int // высота изображения
}

// VariantResult представляет результат обработки одного варианта
type VariantResult struct {
	Type        string // "resized", "thumbnail", "watermarked"
	StoragePath string
	ContentType string
	Size        int64
	Width       *int
	Height      *int
}

// ImageResult представляет полный результат обработки изображения
type ImageResult struct {
	ImageID      string          // UUID оригинала
	Status       string          // "completed" или "failed"
	ErrorMessage *string         // если failed
	Variants     []VariantResult // варианты (если успешно)
}
