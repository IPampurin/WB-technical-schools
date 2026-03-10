package s3

import (
	"context"
	"io"
)

// S3Methods описывает методы для работы с распределённым хранилищем
type S3Methods interface {

	// Upload сохраняет файл в хранилище по указанному ключу (пути)
	Upload(ctx context.Context, path string, reader io.Reader, contentType string) error

	// Download возвращает ReadCloser для чтения файла по указанному ключу (пути)
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete удаляет файл по указанному ключу (пути)
	Delete(ctx context.Context, path string) error

	// GetBucket возвращает имя бакета хранилища
	GetBucket() string
}
