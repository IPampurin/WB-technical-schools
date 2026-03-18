package api

// Item представляет товар на складе
// (GET /items (массив Item), POST /items или PUT /items/{id} (один Item))
type Item struct {
	ID       int     `json:"id"`       // уникальный идентификатор товара
	Name     string  `json:"name"`     // название товара
	Quantity int     `json:"quantity"` // количество на складе
	Price    float64 `json:"price"`    // цена за единицу
}

// CreateUpdateItemRequest — тело запроса для создания нового товара или обновления существующего
// (POST /items, PUT /items/{id})
type CreateUpdateItemRequest struct {
	Name     string  `json:"name"       binding:"required"`       // название товара, обязательно
	Quantity int     `json:"quantity"   binding:"required,min=0"` // количество, неотрицательное
	Price    float64 `json:"price"      binding:"required,min=0"` // цена, неотрицательная
}

// HistoryRecord представляет одну запись в истории изменений товара
// (GET /items/{id}/history (массив HistoryRecord), GET /history)
type HistoryRecord struct {
	ID        int         `json:"id"`                 // уникальный идентификатор записи истории
	ItemID    int         `json:"item_id"`            // идентификатор товара
	Action    string      `json:"action"`             // тип действия: INSERT, UPDATE, DELETE
	OldData   interface{} `json:"old_data,omitempty"` // старые данные (JSON-объект), для UPDATE и DELETE
	NewData   interface{} `json:"new_data,omitempty"` // новые данные (JSON-объект), для INSERT и UPDATE
	ChangedBy *int        `json:"changed_by"`         // идентификатор пользователя, совершившего изменение (может быть null)
	ChangedAt string      `json:"changed_at"`         // дата и время изменения в формате RFC3339
}

// HistoryQuery — параметры фильтрации и пагинации для запроса истории
// (GET /items/{id}/history?..., GET /history?...)
type HistoryQuery struct {
	ItemID   int    `form:"item_id"`                                           // фильтр по товару (для общей истории)
	FromDate string `form:"from_date" time_format:"2006-01-02T15:04:05Z07:00"` // начало периода (RFC3339)
	ToDate   string `form:"to_date"   time_format:"2006-01-02T15:04:05Z07:00"` // конец периода (RFC3339)
	UserID   int    `form:"user_id"`                                           // фильтр по пользователю
	Action   string `form:"action"`                                            // фильтр по действию (INSERT, UPDATE, DELETE)
	Limit    int    `form:"limit,default=100"`                                 // максимальное количество записей
	Offset   int    `form:"offset,default=0"`                                  // смещение для пагинации
}

// DiffResponse — результат сравнения двух версий товара
// (GET /items/{id}/history/compare?from={historyID}&to={historyID})
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

// ExportHistoryQuery — параметры для экспорта истории в CSV
// (GET /export/history/csv?...)
type ExportHistoryQuery struct {
	ItemID   int    `form:"item_id"`                                           // если указан, экспортируется история только этого товара
	FromDate string `form:"from_date" time_format:"2006-01-02T15:04:05Z07:00"` // начало периода (RFC3339)
	ToDate   string `form:"to_date"   time_format:"2006-01-02T15:04:05Z07:00"` // конец периода (RFC3339)
	UserID   int    `form:"user_id"`                                           // фильтр по пользователю
	Action   string `form:"action"`                                            // фильтр по действию
}

// ErrorResponse — стандартный формат ответа с ошибкой
type ErrorResponse struct {
	Error string `json:"error"` // текст ошибки
}

// LoginRequest — тело запроса на вход
type LoginRequest struct {
	Role string `json:"role" binding:"required,oneof=admin manager viewer"` // роль пользователя (admin, manager, viewer)
}

// LoginResponse — ответ сервера на успешный вход
type LoginResponse struct {
	Token string `json:"token"` // JWT-токен для аутентификации
}
