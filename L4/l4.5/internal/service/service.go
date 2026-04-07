package service

import (
	"fmt"
)

type Service struct{}

func NewService() *Service {

	return &Service{}
}

// Sum реализует сложение
func (s *Service) Sum(a, b int64) int64 {

	return a + b
}

// Fib реализует итеративное вычисление чисел Фибоначчи (для n == 0 возвращает 0)
func (s *Service) Fib(n int) int64 {

	if n <= 0 {
		return 0
	}
	if n == 1 {
		return 1
	}

	var a, b int64 = 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}

	return b
}

// Allocate выделяет память и сразу освобождает
func (s *Service) Allocate(size int) (int, error) {

	if size <= 0 {
		return 0, fmt.Errorf("размер должен быть положительным: %d", size)
	}
	// выделяем слайс, чтобы нагрузить GC
	_ = make([]byte, size)

	return size, nil
}
