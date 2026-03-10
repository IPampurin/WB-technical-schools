package service

import (
	"context"
	"io"

	"github.com/IPampurin/ImageProcessor/pkg/domain"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/logger"
)

// ServiceMethods - интерфейс слоя бизнес-логики
type ServiceMethods interface {

	// UploadImage загружает изображение, сохраняет его в S3, создаёт запись в БД и задачу в outbox
	UploadImage(ctx context.Context, data *domain.UploadData, log logger.Logger) (uuid.UUID, error)

	// GetImage возвращает файл изображения по его ID и варианту (original, thumbnail, resized и тд)
	// (возвращает ReadCloser (который нужно закрыть после использования), ContentType и ошибку)
	GetImage(ctx context.Context, id uuid.UUID, variant string, log logger.Logger) (io.ReadCloser, string, error)

	// DeleteImage удаляет изображение и все его обработанные варианты из БД и S3
	DeleteImage(ctx context.Context, id uuid.UUID, log logger.Logger) error

	// ListImages возвращает limit последних загруженных оригинальных изображений для отображения в галерее
	ListImages(ctx context.Context, limit int, log logger.Logger) ([]*domain.ImageData, error)

	// ProcessResult обрабатывает результат из очереди, обновляет БД
	ProcessResult(ctx context.Context, result *domain.ImageResult, log logger.Logger) error

	// GetVariants возвращает все варианты для оригинала
	GetVariants(ctx context.Context, originalID uuid.UUID, log logger.Logger) ([]*domain.ImageData, error)
}
