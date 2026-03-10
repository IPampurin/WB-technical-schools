package operations

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

// AddTextWatermark накладывает полупрозрачный текстовый водяной знак по центру изображения
// (размер шрифта: 10% от ширины, но не менее 20 и не более 80; цвет: чёрный с альфа 180)
func AddTextWatermark(img image.Image, text string) (image.Image, error) {

	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	fontParsed, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}

	// определяем размер шрифта пропорционально ширине
	imgWidth := float64(bounds.Dx())
	fontSize := imgWidth * 0.1
	if fontSize < 20 {
		fontSize = 20
	}
	if fontSize > 80 {
		fontSize = 80
	}

	face := truetype.NewFace(fontParsed, &truetype.Options{
		Size: fontSize,
		DPI:  72,
	})
	defer face.Close()

	// точное измерение ширины текста
	drawer := &font.Drawer{Face: face}
	textWidth := drawer.MeasureString(text).Round()
	textHeight := int(fontSize * 1.2)

	// координаты для центрирования
	x := (bounds.Dx() - textWidth) / 2
	if x < 0 {
		x = 0
	}
	y := (bounds.Dy()-textHeight)/2 + int(fontSize*0.8)

	// рисуем текст
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(fontParsed)
	c.SetFontSize(fontSize)
	c.SetClip(bounds)
	c.SetDst(dst)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 180}))

	pt := freetype.Pt(x, y)
	if _, err := c.DrawString(text, pt); err != nil {
		return nil, err
	}

	return dst, nil
}
