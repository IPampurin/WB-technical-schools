/* ВАРИАНТ №1 - решение задачи l2.2 */

package main

import "fmt"

func test() (x int) {

	defer func() {
		x++
	}()
	x = 1
	return
}

func anotherTest() int {

	var x int
	defer func() {
		x++
	}()
	x = 1
	return x
}

func main() {

	// выводим в консоль значения переданные из функций
	fmt.Println(test())
	fmt.Println(anotherTest())
}
