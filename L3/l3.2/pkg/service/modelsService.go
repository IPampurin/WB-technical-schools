package service

import "time"

// ResponseLink - ответ на успешное создание (POST /shorten выход) или запрос данных (элемент на GET /links выход)
type ResponseLink struct {
	ID          int       `json:"-"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	ClicksCount int       `json:"clicks_count"`
}

// FollowLink - информация об одном переходе (для аналитики)
type FollowLink struct {
	AccessedAt time.Time `json:"accessed_at"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address,omitempty"`
	Referer    string    `json:"referer,omitempty"`
}

// ResponseAnalytics - полный ответ для GET /analytics/:short_url
type ResponseAnalytics struct {
	Link              ResponseLink   `json:"link"`
	Analytics         []FollowLink   `json:"analytics"`
	ClicksByDay       map[string]int `json:"clicks_by_day,omitempty"`
	ClicksByMonth     map[string]int `json:"clicks_by_month,omitempty"`
	ClicksByUserAgent map[string]int `json:"clicks_by_user_agent,omitempty"`
}
