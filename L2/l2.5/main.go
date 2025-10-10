/* ВАРИАНТ №1 - решение задачи l2.5 */

package main

// customError кастомный тип ошибки
type customError struct {
	msg string
}

// Error метод реализующий интерфейс error для типа customError
func (e *customError) Error() string {
	return e.msg
}

// test возвращает указатель на тип customError со значением nil
func test() *customError {
	// ... do something
	return nil
}

func main() {

	// объявляем ошибку err (интерфейс error)
	var err error

	// присваиваем переменной err результат функции test(),
	// то есть значение nil типа *customError
	err = test()

	// пытаемся сравнить с nil переменную интерфейса error,
	// у которой значение nil, а тип - *customError.
	// интерфейс не равен nil, потому что у него есть тип
	if err != nil {
		println("error") // вывод программы: error
		return
	}
	println("ok")
}
