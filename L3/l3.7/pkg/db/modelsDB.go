package db

import (
	"time"
)

// User представляет запись в таблице users
type User struct {
	ID        int       `db:"id"`         // id пользователя
	Name      string    `db:"name"`       // имя пользователя
	Email     string    `db:"email"`      // уникальный email пользователя
	Role      string    `db:"role"`       // admin, manager, viewer
	CreatedAt time.Time `db:"created_at"` // дата создания записи о пользователе
}

// Item представляет запись в таблице items
type Item struct {
	ID        int       `db:"id"`         // id товара
	Name      string    `db:"name"`       // наименование
	Quantity  int       `db:"quantity"`   // количество (неотрицательное)
	Price     float64   `db:"price"`      // цена товара
	CreatedAt time.Time `db:"created_at"` // дата создания записи о товаре
	UpdatedAt time.Time `db:"updated_at"` // дата обновления данных о товаре
}

// ItemHistory представляет запись в таблице item_history
type ItemHistory struct {
	ID        int       `db:"id"`         // id записи об изменениях
	ItemID    int       `db:"item_id"`    // id товара
	Action    string    `db:"action"`     // действие (INSERT, UPDATE, DELETE)
	OldData   []byte    `db:"old_data"`   // предыдущие данные (JSONB как []byte)
	NewData   []byte    `db:"new_data"`   // новые данные (JSONB как []byte)
	ChangedBy *int      `db:"changed_by"` // id пользователя, совершившего изменение (может быть NULL, если пользователь удалён)
	ChangedAt time.Time `db:"changed_at"` // время изменений
}

// HistoryFilter - структура для фильтрации истории, используется в StorageItemsHistory
type HistoryFilter struct {
	ItemID   int       // id товара (0 - все товары)
	FromDate time.Time // начало периода (zero - не учитывается)
	ToDate   time.Time // конец периода (zero - не учитывается)
	UserID   int       // id пользователя (0 - все)
	Action   string    // действие (INSERT, UPDATE, DELETE), пустая строка - все
	Limit    int       // максимальное количество записей (по умолчанию 100)
	Offset   int       // смещение для пагинации
}
