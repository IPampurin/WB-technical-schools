package service

import (
	"math/rand/v2"
)

const sizeShortUrl = 6 // длина сгенерированной короткой ссылки ShortURL по умолчанию

// NewRandomString возвращает случайную строку указанной длины
func NewRandomString(size int) string {

	if size == 0 {
		size = sizeShortUrl
	}

	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789" +
		"+-_!")

	b := make([]rune, size)
	for i := range b {
		b[i] = chars[rand.N(len(chars))]
	}

	return string(b)
}
