package service

import (
	"context"

	"github.com/IPampurin/EventBooker/pkg/db"
	"github.com/IPampurin/EventBooker/pkg/zSet"
)

type Service struct {
	storage db.StorageMethods
	zSet    zSet.ZSetMethods
}

func InitService(ctx context.Context, storage db.StorageMethods, clientZSet zSet.ZSetMethods) *Service {

	return &Service{
		storage: storage,
		zSet:    clientZSet,
	}
}
