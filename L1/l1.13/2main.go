/* ВАРИАНТ №2 - смена чисел с применением сложения/вычитания */

package main

import "fmt"

func main() {

	a, b := 1, 2

	fmt.Printf("\n   До замены: а = %d, b = %d", a, b)

	a = a + b // 1+2=3
	b = a - b // 3-2=1
	a = a - b // 3-1=2

	fmt.Printf("\nПосле замены: а = %d, b = %d\n", a, b)
}
