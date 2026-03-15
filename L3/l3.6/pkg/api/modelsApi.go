package api

// Item представляет запись о продаже/покупке (GET /api/items, POST /api/items, PUT /api/items/{id})
type Item struct {
	ID       int     `json:"id"`       // уникальный идентификатор записи
	Category string  `json:"category"` // категория ("Продукты", "Транспорт", "Развлечения", "Здоровье", "Другое")
	Amount   float64 `json:"amount"`   // сумма
	Date     string  `json:"date"`     // дата в формате YYYY-MM-DD
}

// CreateItemRequest — тело запроса для создания/обновления записи (POST /api/items, PUT /api/items/{id})
type CreateUpdateItemRequest struct {
	Category string  `json:"category" binding:"required"`                  // категория, обязательное поле
	Amount   float64 `json:"amount" binding:"required,min=0"`              // сумма, неотрицательная
	Date     string  `json:"date"  binding:"required,datetime=2006-01-02"` // дата в формате YYYY-MM-DD
}

// AnalyticsResponse содержит общие метрики за период (GET /api/analytics)
type AnalyticsResponse struct {
	Sum          float64 `json:"sum"`          // сумма всех сумм за период
	Avg          float64 `json:"avg"`          // среднее арифметическое
	Count        int     `json:"count"`        // количество записей
	Median       float64 `json:"median"`       // медиана (50-й перцентиль)
	Percentile90 float64 `json:"percentile90"` // 90-й перцентиль
}

// CategoryGroup — агрегированные данные по одной категории (GET /api/analytics/by-category)
type CategoryGroup struct {
	Category string  `json:"category"` // название категории
	Sum      float64 `json:"sum"`      // сумма сумм в этой категории
	Count    int     `json:"count"`    // количество записей в категории
}

// ItemsQuery параметры для GET /api/items
type ItemsQuery struct {
	From      string `form:"from" time_format:"2006-01-02"` // начало периода (YYYY-MM-DD), опционально
	To        string `form:"to"   time_format:"2006-01-02"` // конец периода (YYYY-MM-DD), опционально
	SortBy    string `form:"sort_by"`                       // поле сортировки (date, category, amount)
	SortOrder string `form:"sort_order"`                    // направление (asc, desc)
}

// AnalyticsQuery параметры для GET /api/analytics и GET /api/analytics/by-category
type AnalyticsQuery struct {
	From string `form:"from" binding:"required" time_format:"2006-01-02"` // начало периода, обязательно
	To   string `form:"to"   binding:"required" time_format:"2006-01-02"` // конец периода, обязательно
}

// ExportCSVQuery параметры для GET /api/export/csv
type ExportCSVQuery struct {
	From string `form:"from" binding:"required" time_format:"2006-01-02"`
	To   string `form:"to"   binding:"required" time_format:"2006-01-02"`
}

// ErrorResponse стандартный формат ответа с ошибкой
type ErrorResponse struct {
	Error string `json:"error"` // текст ошибки
}
