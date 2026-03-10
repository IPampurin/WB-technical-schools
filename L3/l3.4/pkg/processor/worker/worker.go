package worker

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"path/filepath"

	"github.com/IPampurin/ImageProcessor/pkg/processor/models"
	"github.com/IPampurin/ImageProcessor/pkg/processor/operations"
	"github.com/IPampurin/ImageProcessor/pkg/s3"
	"github.com/wb-go/wbf/logger"
)

// Worker выполняет операции над изображениями: ресайз, создание миниатюр, наложение водяного знака
type Worker struct {
	s3          *s3.S3
	log         logger.Logger
	thumbWidth  int // ширина миниатюры по умолчанию (из конфига)
	thumbHeight int // высота миниатюры по умолчанию
}

// New создаёт нового воркера с переданными зависимостями
func New(s3 *s3.S3, log logger.Logger, thumbWidth, thumbHeight int) *Worker {

	return &Worker{
		s3:          s3,
		log:         log,
		thumbWidth:  thumbWidth,
		thumbHeight: thumbHeight,
	}
}

// ProcessTask обрабатывает одну задачу: скачивает оригинал из S3, применяет запрошенные операции,
// загружает полученные варианты обратно в S3 и формирует результат
//
// Возвращает Result со статусом "completed", если хотя бы одна операция успешна,
// или "failed", если ни одна операция не удалась (при этом заполняется ErrorMessage).
func (w *Worker) ProcessTask(ctx context.Context, task *models.Task) (*models.Result, error) {

	// 1. Скачиваем оригинал из S3
	reader, err := w.s3.Download(ctx, task.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка скачивания оригинала из S3: %w", err)
	}
	defer reader.Close()

	// 2. Декодируем изображение для определения формата (JPEG/PNG)
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования изображения: %w", err)
	}

	// 3. Определяем MIME-тип по формату
	contentType := "image/jpeg"
	if format == "png" {
		contentType = "image/png"
	}

	var variants []models.VariantResult

	// 4. Ресайз (если запрошен)
	if task.Resize != nil {
		resizedImg, err := operations.Resize(img, task.Resize.Width, task.Resize.Height)
		if err != nil {
			w.log.Error("ошибка ресайза", "error", err, "imageID", task.ImageID)
		} else {
			key := generateKey(task.ImageID, "resized", filepath.Ext(task.ObjectKey))
			size, err := w.uploadImage(ctx, key, resizedImg, contentType)
			if err != nil {
				w.log.Error("ошибка загрузки ресайза в S3", "error", err, "key", key)
			} else {
				width := task.Resize.Width
				height := task.Resize.Height
				variants = append(variants, models.VariantResult{
					Type:        "resized",
					StoragePath: key,
					ContentType: contentType,
					Size:        size,
					Width:       &width,
					Height:      &height,
				})
			}
		}
	}

	// 5. Миниатюра (если запрошена)
	if task.Thumbnail {
		thumbImg, err := operations.Thumbnail(img, w.thumbWidth, w.thumbHeight)
		if err != nil {
			w.log.Error("ошибка создания миниатюры", "error", err, "imageID", task.ImageID)
		} else {
			key := generateKey(task.ImageID, "thumbnail", filepath.Ext(task.ObjectKey))
			size, err := w.uploadImage(ctx, key, thumbImg, contentType)
			if err != nil {
				w.log.Error("ошибка загрузки миниатюры в S3", "error", err, "key", key)
			} else {
				width := w.thumbWidth
				height := w.thumbHeight
				variants = append(variants, models.VariantResult{
					Type:        "thumbnail",
					StoragePath: key,
					ContentType: contentType,
					Size:        size,
					Width:       &width,
					Height:      &height,
				})
			}
		}
	}

	// 6. Водяной знак (если запрошен)
	if task.Watermark {
		// текст водяного знака можно брать из конфига, для примера — фиксированный
		watermarkedImg, err := operations.AddTextWatermark(img, "Watermark")
		if err != nil {
			w.log.Error("ошибка наложения водяного знака", "error", err, "imageID", task.ImageID)
		} else {
			key := generateKey(task.ImageID, "watermarked", filepath.Ext(task.ObjectKey))
			size, err := w.uploadImage(ctx, key, watermarkedImg, contentType)
			if err != nil {
				w.log.Error("ошибка загрузки изображения с водяным знаком в S3", "error", err, "key", key)
			} else {
				bounds := watermarkedImg.Bounds()
				width := bounds.Dx()
				height := bounds.Dy()
				variants = append(variants, models.VariantResult{
					Type:        "watermarked",
					StoragePath: key,
					ContentType: contentType,
					Size:        size,
					Width:       &width,
					Height:      &height,
				})
			}
		}
	}

	// 7. Формируем результат
	result := &models.Result{
		ImageID:  task.ImageID,
		Status:   "completed",
		Variants: variants,
	}

	// если были запрошены операции, но ни одна не удалась — помечаем как failed
	if len(variants) == 0 && (task.Resize != nil || task.Thumbnail || task.Watermark) {
		errMsg := "все операции завершились ошибкой"
		result.Status = "failed"
		result.ErrorMessage = &errMsg
	}

	return result, nil
}

// uploadImage кодирует изображение в JPEG или PNG (в зависимости от contentType) и загружает полученные байты в S3 по указанному ключу.
// Возвращает размер загруженного файла в байтах или ошибку.
func (w *Worker) uploadImage(ctx context.Context, key string, img image.Image, contentType string) (int64, error) {

	var data []byte
	var err error

	// кодируем изображение в зависимости от MIME-типа
	if contentType == "image/jpeg" {
		data, err = operations.EncodeJPEG(img)
	} else {
		data, err = operations.EncodePNG(img)
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка кодирования изображения: %w", err)
	}

	// загружаем в S3
	if err := w.s3.Upload(ctx, key, bytes.NewReader(data), contentType); err != nil {
		return 0, fmt.Errorf("ошибка загрузки в S3: %w", err)
	}

	// возвращаем размер данных
	return int64(len(data)), nil
}

// generateKey формирует ключ для S3 вида: {imageID}/{variant}.ext,
// например: "123e4567-e89b-12d3-a456-426614174000/thumbnail.jpg"
func generateKey(imageID, variant, ext string) string {

	return fmt.Sprintf("%s/%s%s", imageID, variant, ext)
}
