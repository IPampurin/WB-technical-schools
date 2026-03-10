package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/IPampurin/ImageProcessor/pkg/domain"
	"github.com/IPampurin/ImageProcessor/pkg/manager/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/logger"
)

// UploadImageToProcess возвращает gin.HandlerFunc для загрузки изображения
func UploadImageToProcess(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. Получаем файл из формы
		file, header, err := c.Request.FormFile("image")
		if err != nil {
			log.Error("не удалось получить файл", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "файл не найден"})
			return
		}
		defer file.Close()

		// 2. Определяем MIME-тип по первым 512 байтам
		buff := make([]byte, 512)
		n, err := file.Read(buff)
		if err != nil && err != io.EOF {
			log.Error("ошибка чтения первых байт", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось прочитать файл"})
			return
		}
		contentType := http.DetectContentType(buff[:n])

		// 3. Читаем весь файл в буфер (чтобы получить seekable reader)
		//    Сначала нужно собрать полный файл: уже прочитанные 512 байт + остаток
		//    Используем MultiReader для объединения, затем читаем всё в память
		fullReader := io.MultiReader(bytes.NewReader(buff[:n]), file)
		fileBytes, err := io.ReadAll(fullReader)
		if err != nil {
			log.Error("ошибка чтения файла", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось прочитать файл"})
			return
		}

		// 4. Создаём bytes.Reader (он поддерживает Seek)
		reader := bytes.NewReader(fileBytes)

		// 5. Парсим остальные поля формы
		thumbnail := c.PostForm("thumbnail") == "true"
		watermark := c.PostForm("watermark") == "true"

		var resize *domain.ResizeOptions
		widthStr := c.PostForm("width")
		heightStr := c.PostForm("height")

		if widthStr != "" || heightStr != "" {
			if widthStr == "" || heightStr == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "необходимо указать и ширину, и высоту для ресайза"})
				return
			}
			width, err := strconv.Atoi(widthStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "ширина должна быть числом"})
				return
			}
			height, err := strconv.Atoi(heightStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "высота должна быть числом"})
				return
			}
			resize = &domain.ResizeOptions{Width: width, Height: height}
		}

		// 6. Формируем доменную структуру запроса
		uploadData := &domain.UploadData{
			Filename:    header.Filename,
			ContentType: contentType,
			Size:        header.Size, // размер известен из header
			Reader:      reader,      // используем seekable reader
			Thumbnail:   thumbnail,
			Watermark:   watermark,
			Resize:      resize,
		}

		// 7. Вызываем сервис
		id, err := svc.UploadImage(c.Request.Context(), uploadData, log)
		if err != nil {
			log.Error("ошибка загрузки изображения", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 8. Возвращаем UUID
		c.JSON(http.StatusAccepted, UploadResponse{ID: id})
	}
}

// LoadImageFromProcess возвращает файл изображения по ID и варианту
func LoadImageFromProcess(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный ID"})
			return
		}

		variant := c.DefaultQuery("variant", "original")

		reader, contentType, err := svc.GetImage(c.Request.Context(), id, variant, log)
		if err != nil {
			log.Error("ошибка получения изображения", "error", err, "id", id, "variant", variant)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer reader.Close()

		// формируем имя файла для скачивания
		filename := fmt.Sprintf("%s_%s", id.String(), variant)
		switch contentType {
		case "image/jpeg":
			filename += ".jpg"
		case "image/png":
			filename += ".png"
		default:
			filename += ".bin"
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
	}
}

// DeleteImage удаляет изображение и все его варианты
func DeleteImage(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный ID"})
			return
		}

		if err := svc.DeleteImage(c.Request.Context(), id, log); err != nil {
			log.Error("ошибка удаления изображения", "error", err, "id", id)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	}
}

// GetImages возвращает список последних загруженных изображений
func GetImages(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. Получаем limit из query (по умолчанию 20)
		limitStr := c.DefaultQuery("limit", "20")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 20
		}

		// 2. Вызываем сервис (получаем доменные модели)
		originals, err := svc.ListImages(c.Request.Context(), limit, log)
		if err != nil {
			log.Error("ошибка получения списка изображений", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 3. Преобразуем в ответ для фронта
		response := make([]imageResponse, 0, len(originals))
		for _, orig := range originals {
			variants, err := svc.GetVariants(c.Request.Context(), orig.ID, log)
			if err != nil {
				log.Error("ошибка получения вариантов", "error", err, "originalID", orig.ID)
				// продолжаем, варианты будут пустыми
			}

			variantResponses := make([]imageVariantResponse, 0, len(variants))
			for _, v := range variants {
				variantResponses = append(variantResponses, imageVariantResponse{
					Type:   v.Type,
					Width:  v.Width,
					Height: v.Height,
					Size:   v.Size,
				})
			}

			response = append(response, imageResponse{
				ID:           orig.ID,
				OriginalName: orig.Name,
				Status:       orig.Status,
				Variants:     variantResponses,
			})
		}

		c.JSON(http.StatusOK, response)
	}
}
