/* ВАРИАНТ №1 - решение задачи l1.24 */

package main

import (
	"fmt"
	"math"
)

// Point описывает точку на плоскости
type Point struct {
	x float64
	y float64
}

// NewPoint возвращает указатель на структуру Point с указанными параметрами
func NewPoint(x, y float64) *Point {

	return &Point{x: x, y: y}
}

// Distance находит расстояние между точками на плоскости
func (some *Point) Distance(other *Point) float64 {

	dx := math.Pow((other.x - some.x), 2)
	dy := math.Pow((other.y - some.y), 2)
	distance := math.Sqrt(dx + dy)

	return distance
}

func main() {

	a := NewPoint(0, 0)  // указатель на одну точку
	b := NewPoint(10, 0) // указатель на другую точку

	fmt.Printf("Расстояние между точками a и b: distance = %.2f ед.", a.Distance(b))
}
