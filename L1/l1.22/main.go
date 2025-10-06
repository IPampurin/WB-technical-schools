/* ВАРИАНТ №1 - решение задачи l1.22 */

package main

import (
	"fmt"
	"math/big"
)

func main() {

	/*
		// для чисел больше int64
		// создаем большие числа через строки
			a := new(big.Int)
			a.SetString("123456789012345678901234567890", 10) // основание системы счисления 10
			b := new(big.Int)
			b.SetString("987654321098765432109876543210", 10) // основание системы счисления 10
	*/

	// создаём большие числа через big.NewInt (для чисел меньше int64)
	a := big.NewInt(10_000_000)
	b := big.NewInt(10_000_000)

	// выполняем сложение
	c := new(big.Int).Add(a, b)
	fmt.Println("Результат сложения a + b =", c)

	// вычисляем разность
	d := new(big.Int).Sub(a, b)
	fmt.Println("Результат вычитания a - b =", d)

	// вычисляем произведение
	e := new(big.Int).Mul(a, b)
	fmt.Println("Результат умножения a * b =", e)

	// выполняем деление, если можем
	f := new(big.Int)
	if b.Sign() == 0 { // Sign returns: -1 if x < 0; 0 if x == 0; +1 if x > 0.
		fmt.Println("Результат деления a : b не определён (b = 0)")
	} else {
		f.Div(a, b)
		fmt.Println("Результат деления a : b =", f)
	}
}
