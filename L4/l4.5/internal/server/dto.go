package server

// SumRequest параметры GET /sum?a=1&b=2
type SumRequest struct {
	A *int64 `form:"a"`
	B *int64 `form:"b"`
}

// FibRequest параметры GET /fib?n=40
type FibRequest struct {
	N int `form:"n" binding:"required,min=1,max=45"` // ограничим 45 для приемлемого времени
}

// AllocateRequest тело POST /allocate
type AllocateRequest struct {
	Size int `json:"size" binding:"required,min=1,max=100000000"` // до 100 млн байт
}

// SuccessResponse стандартный успешный ответ
type SuccessResponse struct {
	Result interface{} `json:"result"`
}

// ErrorResponse ответ с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}
