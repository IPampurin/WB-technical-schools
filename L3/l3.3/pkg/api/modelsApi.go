package api

// CreateRequest - запрос на создание короткой ссылки (POST /shorten вход)
type CreateRequest struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
	CustomShort string `json:"custom_short" binding:"omitempty,alphanum,max=50"`
}

// ErrorResponse - стандартный ответ с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}
