package zSet

import "context"

// ZSetMethods - методы, которые должны быть у реализации ZSET
type ZSetMethods interface {

	// ZRangeByScore получает элементы из сортированного множества с баллами в интервале [min, max]
	ZRangeByScore(ctx context.Context, min, max int64) ([]string, error)

	// ZRem удаляет элемент из сортированного множества
	ZRem(ctx context.Context, members ...interface{}) error

	// ZAdd добавляет элемент в сортированное множество с указанным счётом (score)
	ZAdd(ctx context.Context, score float64, member interface{}) error
}
