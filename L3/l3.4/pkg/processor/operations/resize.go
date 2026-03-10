package operations

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
)

// Resize изменяет размер изображения до указанной ширины и высоты.
// Если ширина или высота меньше или равны нулю, возвращается ошибка.
//
// Параметры:
//   - img: исходное изображение
//   - width:  новая ширина в пикселях (должна быть > 0)
//   - height: новая высота в пикселях (должна быть > 0)
//
// Возвращает:
//   - новое изображение с заданными размерами
//   - ошибку, если переданные размеры некорректны
func Resize(img image.Image, width, height int) (image.Image, error) {

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("некорректные размеры: %dx%d (должны быть положительными)", width, height)
	}

	return imaging.Resize(img, width, height, imaging.Lanczos), nil
}

// EncodeJPEG кодирует изображение в формат JPEG и возвращает срез байт
// (качество сжатия установлено в 85 (баланс между размером и качеством))
//
// Параметры:
//   - img: изображение для кодирования
//
// Возвращает:
//   - байтовое представление JPEG-изображения
//   - ошибку, если кодирование не удалось
func EncodeJPEG(img image.Image) ([]byte, error) {

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})

	return buf.Bytes(), err
}

// EncodePNG кодирует изображение в формат PNG и возвращает срез байт
// (используются настройки сжатия по умолчанию (уровень 6))
//
// Параметры:
//   - img: изображение для кодирования
//
// Возвращает:
//   - байтовое представление PNG-изображения
//   - ошибку, если кодирование не удалось
func EncodePNG(img image.Image) ([]byte, error) {

	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)

	return buf.Bytes(), err
}
