package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/IPampurin/ImageProcessor/pkg/domain"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/logger"
)

// UploadImage загружает изображение в S3, создаёт запись в БД и задачу в outbox
func (s *Service) UploadImage(ctx context.Context, data *domain.UploadData, log logger.Logger) (uuid.UUID, error) {

	// 1. Генерируем уникальный идентификатор для изображения
	imageID := uuid.New()

	// 2. Определяем расширение файла для ключа в S3
	ext := filepath.Ext(data.Filename)
	if ext == "" {
		// если расширение отсутствует, определяем по MIME-типу
		switch data.ContentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		default:
			return uuid.Nil, fmt.Errorf("неподдерживаемый формат файла")
		}
	}
	storagePath := fmt.Sprintf("originals/%s%s", imageID.String(), ext)

	// 3. Загружаем файл в S3 напрямую из reader (потоково)
	if err := s.s3.Upload(ctx, storagePath, data.Reader, data.ContentType); err != nil {
		return uuid.Nil, fmt.Errorf("ошибка загрузки в S3: %w", err)
	}

	// 4. Подготавливаем данные для вставки в таблицу images (оригинал)
	now := time.Now()
	imgData := &domain.ImageData{
		ID:           imageID,
		OriginalID:   nil, // это оригинал
		Name:         data.Filename,
		Type:         "original",
		ContentType:  data.ContentType,
		Size:         data.Size,
		Width:        nil, // будут заполнены после обработки
		Height:       nil,
		Status:       "pending", // начальный статус
		ErrorMessage: nil,
		StoragePath:  storagePath,
		CreatedAt:    now,
	}

	// 5. Сохраняем запись в БД
	if err := s.image.InsertImage(ctx, imgData); err != nil {
		// если не удалось сохранить в БД — удаляем загруженный файл из S3
		if delErr := s.s3.Delete(ctx, storagePath); delErr != nil {
			log.Error("не удалось удалить файл из S3 после ошибки вставки в БД", "error", delErr, "storagePath", storagePath)
		}
		return uuid.Nil, fmt.Errorf("ошибка сохранения записи в БД: %w", err)
	}

	// 6. Формируем задачу для отправки в очередь
	task := domain.ImageTask{
		ImageID:      imageID.String(),
		ObjectKey:    storagePath,
		Bucket:       s.s3.GetBucket(), // имя бакета из S3-клиента
		Thumbnail:    data.Thumbnail,
		Watermark:    data.Watermark,
		Resize:       nil,
		OriginalName: data.Filename,
	}
	if data.Resize != nil {
		task.Resize = &domain.ResizeOptions{
			Width:  data.Resize.Width,
			Height: data.Resize.Height,
		}
	}

	// сериализуем задачу в JSON
	payload, err := json.Marshal(task)
	if err != nil {
		// ошибка маршалинга маловероятна, но откатываем изменения
		if delErr := s.image.DeleteImage(ctx, imageID); delErr != nil {
			log.Error("не удалось удалить запись из БД после ошибки маршалинга", "error", delErr, "imageID", imageID)
		}
		if delErr := s.s3.Delete(ctx, storagePath); delErr != nil {
			log.Error("не удалось удалить файл из S3 после ошибки маршалинга", "error", delErr, "storagePath", storagePath)
		}
		return uuid.Nil, fmt.Errorf("ошибка формирования задачи: %w", err)
	}

	// 7. Создаём запись в outbox
	outboxData := &domain.OutboxData{
		ID:        uuid.New(),
		Topic:     s.inputTopic,
		Key:       imageID.String(),
		Payload:   payload,
		CreatedAt: now,
	}

	if err := s.outbox.CreateOutbox(ctx, outboxData); err != nil {
		// ошибка сохранения в outbox — откатываем БД и S3
		if delErr := s.image.DeleteImage(ctx, imageID); delErr != nil {
			log.Error("не удалось удалить файл из БД после ошибки вставки в outbox", "error", delErr, "storagePath", storagePath)
		}
		if delErr := s.s3.Delete(ctx, storagePath); delErr != nil {
			log.Error("не удалось удалить файл из S3 после ошибки вставки в outbox", "error", delErr, "storagePath", storagePath)
		}
		return uuid.Nil, fmt.Errorf("ошибка сохранения задачи в outbox: %w", err)
	}

	// 8. Логируем успех
	log.Info("изображение успешно загружено", "imageID", imageID, "storagePath", storagePath)

	return imageID, nil
}

// GetImage возвращает файл изображения по его ID и варианту (original, thumbnail, resized и т.д.)
// (возвращает ReadCloser (нужно закрыть после использования), ContentType и ошибку)
func (s *Service) GetImage(ctx context.Context, id uuid.UUID, variant string, log logger.Logger) (io.ReadCloser, string, error) {

	// 1. Получаем запись из БД по ID
	img, err := s.image.GetByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("изображение не найдено: %w", err)
	}

	// 2. Определяем, какой вариант запрошен
	var targetImg *domain.ImageData
	if variant == "original" || variant == "" {
		targetImg = img
	} else {
		// если запрошен не оригинал, ищем среди вариантов по original_id и type
		variants, err := s.image.GetVariantsByOriginalID(ctx, id)
		if err != nil {
			return nil, "", fmt.Errorf("не удалось получить варианты: %w", err)
		}
		for _, v := range variants {
			if v.Type == variant {
				targetImg = v
				break
			}
		}
		if targetImg == nil {
			return nil, "", fmt.Errorf("вариант %s не найден", variant)
		}

		// для вариантов проверяем статус (оригинал доступен всегда)
		if targetImg.Status != "completed" {
			return nil, "", fmt.Errorf("вариант ещё не обработан (статус: %s)", targetImg.Status)
		}
	}

	// 3. Скачиваем файл из S3 по StoragePath
	reader, err := s.s3.Download(ctx, targetImg.StoragePath)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка загрузки файла из S3: %w", err)
	}

	// 4. Возвращаем ридер и ContentType
	return reader, targetImg.ContentType, nil
}

// DeleteImage удаляет изображение и все его обработанные варианты из БД и S3
func (s *Service) DeleteImage(ctx context.Context, id uuid.UUID, log logger.Logger) error {

	// 1. Получаем запись оригинального изображения
	original, err := s.image.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("изображение не найдено: %w", err)
	}

	// 2. Получаем все варианты этого оригинала
	variants, err := s.image.GetVariantsByOriginalID(ctx, id)
	if err != nil {
		log.Error("не удалось получить варианты для удаления", "error", err, "originalID", id)
		// продолжаем, чтобы удалить хотя бы оригинал
	}

	// 3. Собираем пути для удаления из S3 (оригинал + варианты)
	pathsToDelete := []string{original.StoragePath}
	for _, v := range variants {
		pathsToDelete = append(pathsToDelete, v.StoragePath)
	}

	// 4. Удаляем файлы из S3 (игнорируем ошибки, чтобы попытаться удалить остальные)
	for _, path := range pathsToDelete {
		if delErr := s.s3.Delete(ctx, path); delErr != nil {
			log.Error("не удалось удалить файл из S3", "error", delErr, "path", path)
		}
	}

	// 5. Удаляем запись из БД (каскадно удалятся и варианты, если настроено)
	if err := s.image.DeleteImage(ctx, id); err != nil {
		return fmt.Errorf("ошибка удаления записи из БД: %w", err)
	}

	return nil
}

// ListImages возвращает список последних загруженных оригинальных изображений для отображения в галерее
func (s *Service) ListImages(ctx context.Context, limit int, log logger.Logger) ([]*domain.ImageData, error) {

	if limit <= 0 {
		limit = 20 // значение по умолчанию
	}

	images, err := s.image.ListLatestOriginals(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка изображений: %w", err)
	}

	return images, nil
}

// ProcessResult обрабатывает результат из очереди, обновляет БД
func (s *Service) ProcessResult(ctx context.Context, result *domain.ImageResult, log logger.Logger) error {

	imageID, err := uuid.Parse(result.ImageID)
	if err != nil {
		return fmt.Errorf("некорректный imageID: %w", err)
	}

	original, err := s.image.GetByID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("оригинал не найден: %w", err)
	}

	if result.Status == "failed" {
		errMsg := result.ErrorMessage
		if errMsg == nil {
			msg := "unknown error"
			errMsg = &msg
		}
		if err := s.image.UpdateStatusOrErr(ctx, imageID, "failed", errMsg); err != nil {
			return fmt.Errorf("не удалось обновить статус оригинала: %w", err)
		}
		return nil
	}

	// Status == "completed"
	for _, v := range result.Variants {

		variantID := uuid.New()

		var width, height *int
		if v.Width != nil {
			width = v.Width
		}
		if v.Height != nil {
			height = v.Height
		}

		variantData := &domain.ImageData{
			ID:           variantID,
			OriginalID:   &imageID,
			Name:         original.Name, // можно модифицировать
			Type:         v.Type,
			ContentType:  v.ContentType,
			Size:         v.Size,
			Width:        width,
			Height:       height,
			Status:       "completed",
			ErrorMessage: nil,
			StoragePath:  v.StoragePath,
			CreatedAt:    time.Now(),
		}
		if err := s.image.InsertImage(ctx, variantData); err != nil {
			log.Error("не удалось вставить вариант", "error", err, "originalID", imageID, "type", v.Type)
			// продолжаем, но логируем
		}
	}

	// обновляем статус оригинала на completed
	if original.Status != "completed" {
		if err := s.image.UpdateStatusOrErr(ctx, imageID, "completed", nil); err != nil {
			log.Error("не удалось обновить статус оригинала", "error", err, "originalID", imageID)
		}
	}

	return nil
}

// GetVariants возвращает все варианты для оригинала
func (s *Service) GetVariants(ctx context.Context, originalID uuid.UUID, log logger.Logger) ([]*domain.ImageData, error) {

	return s.image.GetVariantsByOriginalID(ctx, originalID)
}
