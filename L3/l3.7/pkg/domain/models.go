package domain

import (
	"errors"
	"time"
)

var (
	// ErrNotFound возвращается, когда запрашиваемый ресурс не найден
	ErrNotFound = errors.New("resource not found")

	// ErrForbidden возвращается при недостаточных правах доступа
	ErrForbidden = errors.New("access forbidden")

	// ErrInvalidInput возвращается при невалидных данных запроса
	ErrInvalidInput = errors.New("invalid input data")
)

// User представляет пользователя системы
type User struct {
	ID        int       // идентификатор пользователя
	Name      string    // имя пользователя
	Email     string    // уникальный email пользователя
	Role      string    // роль: admin, manager, viewer
	CreatedAt time.Time // дата регистрации пользователя
}

// Item представляет товар на складе
type Item struct {
	ID        int       // идентификатор товара
	Name      string    // название товара
	Quantity  int       // количество (неотрицательное)
	Price     float64   // цена товара
	CreatedAt time.Time // дата создания записи о товаре
	UpdatedAt time.Time // дата обновления данных о товаре
}

// ItemHistory представляет запись в истории изменений товара
type ItemHistory struct {
	ID        int       // идентификатор записи истории
	ItemID    int       // идентификатор товара
	Action    string    // действие (INSERT, UPDATE, DELETE)
	OldData   []byte    // предыдущее состояние товара в формате JSON (может быть nil)
	NewData   []byte    // новое состояние товара в формате JSON (может быть nil)
	ChangedBy *int      // идентификатор пользователя, совершившего изменение (nil если пользователь удалён)
	ChangedAt time.Time // время изменений
}

// HistoryFilter содержит параметры фильтрации при запросе истории
// (используется для выборки записей)
type HistoryFilter struct {
	ItemID   int       // если не 0, то только для указанного товара
	FromDate time.Time // если не zero, то только записи после этой даты
	ToDate   time.Time // если не zero, то только записи до этой даты
	UserID   int       // если не 0, то только изменения, сделанные этим пользователем
	Action   string    // если не пустая, то только указанное действие (INSERT, UPDATE, DELETE)
	Limit    int       // максимальное количество записей (0 - использовать значение по умолчанию)
	Offset   int       // смещение для пагинации
}

// DiffResponse — результат сравнения двух версий товара
type DiffResponse struct {
	FromID  int           `json:"from_id"` // ID первой версии (источник)
	ToID    int           `json:"to_id"`   // ID второй версии (цель)
	Changes []FieldChange `json:"changes"` // список изменённых полей
}

// FieldChange описывает изменение одного поля между двумя версиями
type FieldChange struct {
	Field    string      `json:"field"`               // название поля (name, quantity, price)
	OldValue interface{} `json:"old_value,omitempty"` // значение в первой версии (может отсутствовать при создании)
	NewValue interface{} `json:"new_value,omitempty"` // значение во второй версии (может отсутствовать при удалении)
}
