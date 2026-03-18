package service

import (
	"context"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"
)

// ServiceMethods определяет методы бизнес-логики для работы с товарами и историей
type ServiceMethods interface {

	// товары

	// CreateItem создаёт запись о новом товаре
	// Ошибки:
	//   - domain.ErrForbidden - недостаточно прав (не admin/manager)
	//   - domain.ErrInvalidInput - невалидные данные (например, отрицательная цена)
	//   - ошибки БД
	CreateItem(ctx context.Context, item *domain.Item) (*domain.Item, error)

	// GetItemByID возвращает товар по его идентификатору (если товар не найден, возвращает nil, nil)
	// Ошибки:
	//   - domain.ErrForbidden - недостаточно прав (невероятно, но факт)
	//   - ошибки БД
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)

	// GetAllItems возвращает список всех товаров
	// Ошибки:
	//   - domain.ErrForbidden - недостаточно прав (невероятно, но факт)
	//   - ошибки БД
	GetAllItems(ctx context.Context) ([]*domain.Item, error)

	// UpdateItem обновляет существующий товар
	// Ошибки:
	//   - domain.ErrNotFound - товар с таким id не существует
	//   - domain.ErrForbidden - недостаточно прав (не admin/manager)
	//   - domain.ErrInvalidInput - невалидные данные
	//   - ошибки БД
	UpdateItem(ctx context.Context, item *domain.Item) (*domain.Item, error)

	// DeleteItem удаляет товар по id
	// Ошибки:
	//   - domain.ErrNotFound — товар не найден
	//   - domain.ErrForbidden — недостаточно прав (не admin)
	//   - ошибки БД
	DeleteItem(ctx context.Context, id int) error

	// история

	// GetItemHistory возвращает историю изменений для конкретного товара (с фильтрацией и пагинацией)
	// Ошибки:
	//   - domain.ErrForbidden - недостаточно прав (не admin)
	//   - ошибки БД
	GetItemHistory(ctx context.Context, itemID int, filter *domain.HistoryFilter) ([]*domain.ItemHistory, error)

	// GetGlobalHistory возвращает общую историю изменений всех товаров (с фильтрацией и пагинацией)
	// Ошибки:
	//   - domain.ErrForbidden - недостаточно прав (не admin)
	//   - ошибки БД
	GetGlobalHistory(ctx context.Context, filter *domain.HistoryFilter) ([]*domain.ItemHistory, error)

	// GetHistoryByID возвращает одну запись истории по её id (для сравнения версий)
	// (если запись не найдена, возвращает nil, nil)
	// Ошибки:
	//   - domain.ErrForbidden - недостаточно прав (не admin)
	//   - ошибки БД
	GetHistoryByID(ctx context.Context, id int) (*domain.ItemHistory, error)

	// CompareHistoryVersions сравнивает две записи истории и возвращает структуру с различиями
	// Ошибки:
	//   - domain.ErrNotFound — одна из записей не существует
	//   - domain.ErrForbidden — недостаточно прав (не admin)
	//   - ошибки БД или логики сравнения
	CompareHistoryVersions(ctx context.Context, id1, id2 int) (*domain.DiffResponse, error)

	// экспорт

	// ExportItemsCSV генерирует CSV-файл со всеми товарами
	// Ошибки:
	//   - domain.ErrForbidden — недостаточно прав (невероятно, но факт)
	//   - ошибки БД или генерации CSV
	ExportItemsCSV(ctx context.Context) ([]byte, error)

	// ExportHistoryCSV генерирует CSV-файл с записями истории, отфильтрованными по заданным критериям
	// (лимит и оффсет не применяются, выгружаются все записи, удовлетворяющие фильтру)
	// Ошибки:
	//   - domain.ErrForbidden — недостаточно прав (не admin)
	//   - ошибки БД или генерации CSV
	ExportHistoryCSV(ctx context.Context, filter *domain.HistoryFilter) ([]byte, error)
}
