package service

import (
	"context"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/db"
)

type Service struct {
	storage db.Storage
}

func InitService(ctx context.Context, storage db.Storage) *Service {

	return &Service{
		storage: storage,
	}
}
