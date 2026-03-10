package service

import (
	"context"

	"github.com/IPampurin/ImageProcessor/pkg/manager/db"
	"github.com/IPampurin/ImageProcessor/pkg/s3"
)

// Service реализует бизнес-логику приложения
type Service struct {
	image      db.ImageFileMethods
	outbox     db.OutboxMethods
	s3         s3.S3Methods
	inputTopic string // для использования в UploadImage
}

// InitService создаёт новый экземпляр сервиса с переданными зависимостями
func InitService(ctx context.Context, storage *db.DataBase, storageS3 *s3.S3, inputTopic string) *Service {

	svc := &Service{
		image:      storage,   // *db.DataBase реализует ImageFileMethods
		outbox:     storage,   // *db.DataBase реализует OutboxMethods
		s3:         storageS3, // *s3.S3 реализует S3Methods
		inputTopic: inputTopic,
	}

	return svc
}
