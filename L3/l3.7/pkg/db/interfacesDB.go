package db

import (
	"context"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"
)

// Storage - интерфейсы БД
type Storage interface {
	StorageUsers
	StorageItems
	StorageItemsHistory
}

// StorageUsers определяет методы работы с таблицей users
type StorageUsers interface {

	// CreateUser создаёт нового пользователя
	CreateUser(ctx context.Context, user *domain.User) error

	// GetUserByID возвращает пользователя по его ID
	GetUserByID(ctx context.Context, id int) (*domain.User, error)

	// GetUserByEmail возвращает пользователя по email (для аутентификации)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)

	// UpdateUser обновляет данные пользователя (имя, роль)
	UpdateUser(ctx context.Context, user *domain.User) error

	// DeleteUser удаляет пользователя по ID
	DeleteUser(ctx context.Context, id int) error
}

// StorageItems определяет методы работы с таблицей items
type StorageItems interface {

	// CreateItem создаёт новую запись о товаре
	CreateItem(ctx context.Context, item *domain.Item) error

	// GetItemByID возвращает товар по его ID
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)

	// GetAllItems возвращает список всех товаров
	GetAllItems(ctx context.Context) ([]*domain.Item, error)

	// UpdateItem обновляет существующий товар
	UpdateItem(ctx context.Context, item *domain.Item) error

	// DeleteItem удаляет товар по ID
	DeleteItem(ctx context.Context, id int) error
}

// StorageItemsHistory определяет методы работы с таблицей item_history
type StorageItemsHistory interface {

	// GetHistory возвращает записи истории с фильтром и пагинацией,
	// параметры filter позволяют задать item_id, период, пользователя, действие, лимит и смещение
	GetHistory(ctx context.Context, filter *domain.HistoryFilter) ([]*domain.ItemHistory, error)

	// GetHistoryByID возвращает одну запись истории по её ID
	GetHistoryByID(ctx context.Context, id int) (*domain.ItemHistory, error)

	// GetHistoryCount возвращает общее количество записей, удовлетворяющих фильтру (для пагинации)
	GetHistoryCount(ctx context.Context, filter *domain.HistoryFilter) (int, error)
}
