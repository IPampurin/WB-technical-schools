package api

import (
	"mime/multipart"

	"github.com/google/uuid"
)

// приходит с фронта

// ResizeOptions описывает размеры изображения при resize
type ResizeOptions struct {
	Width  int `form:"width"`  // ширина изображения
	Height int `form:"height"` // высота изображения
}

// UploadRequest - структура для парсинга запроса от фронтэнда (POST /upload)
type UploadRequest struct {
	// файл из поля "image"
	File       multipart.File
	FileHeader *multipart.FileHeader
	// чекбоксы (наличие поля означает true)
	Thumbnail bool `form:"thumbnail"`
	Watermark bool `form:"watermark"`
	// если resize присутствует, поле не равно nil и содержит width/height
	Resize *ResizeOptions
}

// уходит на фронт

// UploadResponse возвращается после успешной загрузки изображения на обработку (POST /upload)
type UploadResponse struct {
	ID uuid.UUID `json:"id"` // уникальный идентификатор загруженного изображения
}

// вспомогательные структуры для ответа на /images
type imageVariantResponse struct {
	Type   string `json:"type"`
	Width  *int   `json:"width,omitempty"`
	Height *int   `json:"height,omitempty"`
	Size   int64  `json:"size"`
}

type imageResponse struct {
	ID           uuid.UUID              `json:"id"`
	OriginalName string                 `json:"originalName"`
	Status       string                 `json:"status"`
	Variants     []imageVariantResponse `json:"variants"`
}
