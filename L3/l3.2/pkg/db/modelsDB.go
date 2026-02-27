package db

import (
	"net"
	"time"
)

// Link представляет запись в таблице links
type Link struct {
	ID          int       // внутренний идентификатор ссылки (автоинкремент)
	ShortURL    string    // короткий идентификатор (например, "abc123"), уникален в пределах таблицы
	OriginalURL string    // исходный длинный URL
	CreatedAt   time.Time // дата и время создания записи
	IsCustom    bool      // флаг, указывающий, что short_url задан пользователем
	ClicksCount int       // количество переходов по ссылке (чтобы всё время COUNT не делать)
}

// Analytics представляет запись о переходе по короткой ссылке
type Analytics struct {
	ID         int       // уникальный идентификатор записи о переходе (автоинкремент)
	LinkID     int       // идентификатор ссылки, по которой совершён переход
	AccessedAt time.Time // момент времени, когда произошёл переход
	UserAgent  string    // строка User-Agent браузера или клиента
	IPAddress  net.IP    // IP-адрес посетителя
	Referer    string    // URL источника перехода
}
