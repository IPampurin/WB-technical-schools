package zSet

import (
	"context"
	"strconv"

	goredis "github.com/go-redis/redis/v8"
)

// ZAdd добавляет элемент в сортированное множество с указанным счётом (score)
func (c *ClientZSet) ZAdd(ctx context.Context, score float64, member interface{}) error {

	return c.Client.ZAdd(ctx, c.key, &goredis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// ZRangeByScore получает элементы из сортированного множества с баллами в интервале [min, max]
func (c *ClientZSet) ZRangeByScore(ctx context.Context, min, max int64) ([]string, error) {

	// используем оригинальный тип goredis.ZRangeBy
	return c.Client.ZRangeByScore(ctx, c.key, &goredis.ZRangeBy{
		Min: strconv.FormatInt(min, 10),
		Max: strconv.FormatInt(max, 10),
	}).Result()
}

// ZRem удаляет элемент из сортированного множества
func (c *ClientZSet) ZRem(ctx context.Context, members ...interface{}) error {

	return c.Client.ZRem(ctx, c.key, members...).Err()
}
