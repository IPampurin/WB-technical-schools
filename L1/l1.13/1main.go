/* ВАРИАНТ №1 - идеоматическое решение */

package main

import "fmt"

func main() {

	a, b := 1, 2

	fmt.Printf("\n   До замены: а = %d, b = %d", a, b)

	a, b = b, a

	fmt.Printf("\nПосле замены: а = %d, b = %d\n", a, b)
}
