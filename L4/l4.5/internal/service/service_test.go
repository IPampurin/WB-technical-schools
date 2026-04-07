package service

import "testing"

// BenchmarkSum - бенчмарк функции Sum (лёгкая операция)
func BenchmarkSum(b *testing.B) {

	s := NewService()
	a, bVal := int64(12345), int64(67890)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Sum(a, bVal)
	}
}

// benchmarkFib используется в бенчмарках функции Fib (CPU-нагрузка)
func benchmarkFib(b *testing.B, n int) {

	s := NewService()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Fib(n)
	}
}

// BenchmarkFibN20, BenchmarkFibN30, BenchmarkFibN40 бенчмарки на разные размерности
func BenchmarkFibN20(b *testing.B) { benchmarkFib(b, 20) }
func BenchmarkFibN30(b *testing.B) { benchmarkFib(b, 30) }
func BenchmarkFibN40(b *testing.B) { benchmarkFib(b, 40) }

// benchmarkAllocate используется в бенчмарках функции Allocate (нагрузка на память / GC)
func benchmarkAllocate(b *testing.B, size int) {

	s := NewService()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = s.Allocate(size)
	}
}

// BenchmarkAllocate1MB, BenchmarkAllocate10MB, BenchmarkAllocate100MB бенчмарки на разные размерности
func BenchmarkAllocate1MB(b *testing.B)   { benchmarkAllocate(b, 1<<20) }
func BenchmarkAllocate10MB(b *testing.B)  { benchmarkAllocate(b, 10<<20) }
func BenchmarkAllocate100MB(b *testing.B) { benchmarkAllocate(b, 100<<20) }
