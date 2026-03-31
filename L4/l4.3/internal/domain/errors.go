// доменные ошибки для маппинга в HTTP-коды
package domain

import "errors"

// ErrNotFound - запись отсутствует или не принадлежит пользователю
var ErrNotFound = errors.New("не найдено")
