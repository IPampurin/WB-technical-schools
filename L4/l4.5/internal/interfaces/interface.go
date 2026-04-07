package interfaces

// ServiceMethods - порт бизнес-логики
type ServiceMethods interface {

	// Sum возвращает сумму двух чисел
	Sum(a, b int64) int64

	// Fib вычисляет n-е число Фибоначчи (нагрузка на CPU) (для n <= 0 возвращает 0)
	Fib(n int) int64

	// Allocate выделяет слайс байт заданного размера и сразу освобождает его,
	// возвращает количество выделенных байт или ошибку, если размер некорректен
	Allocate(size int) (int, error)
}
