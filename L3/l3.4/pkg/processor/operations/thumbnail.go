package operations

import (
	"fmt"
	"image"

	"github.com/disintegration/imaging"
)

// Thumbnail создаёт миниатюру изображения, вписывая её в прямоугольник указанных размеров с сохранением пропорций
//
// Параметры:
//   - img: исходное изображение
//   - width:  максимальная ширина миниатюры в пикселях (должна быть > 0)
//   - height: максимальная высота миниатюры в пикселях (должна быть > 0)
//
// Возвращает:
//   - новое изображение-миниатюру
//   - ошибку, если переданные размеры некорректны (меньше или равны нулю)
func Thumbnail(img image.Image, width, height int) (image.Image, error) {

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("некорректные размеры миниатюры: %dx%d (должны быть положительными)", width, height)
	}

	// Fit сохраняет пропорции и вписывает изображение в прямоугольник
	return imaging.Fit(img, width, height, imaging.Lanczos), nil
}
