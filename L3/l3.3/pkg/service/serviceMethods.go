package service

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/UrlShortener/pkg/db"
	"github.com/wb-go/wbf/logger"
)

// CreateShortLink создаёт новую короткую ссылку
// (если customShort не пуст, проверяет его уникальность,
// если оригинальный URL уже существует, возвращает последнюю созданную ссылку,
// в противном случае генерирует случайный shortURL и сохраняет ссылку в БД и кэш)
func (s *Service) CreateShortLink(ctx context.Context, log logger.Logger, originalURL, customUrl string) (*ResponseLink, error) {

	// 1. Если задан кастомный short, проверяем уникальность
	if customUrl != "" {
		existing, err := s.link.GetLinkByShortURL(ctx, customUrl)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, fmt.Errorf("короткая ссылка уже занята")
		}
	}

	// 2. Проверяем, есть ли уже такая оригинальная ссылка
	links, err := s.link.GetLinkByOriginalURL(ctx, originalURL)
	if err != nil {
		return nil, err
	}
	if len(links) > 0 {
		// выбираем последнюю созданную ссылку
		latest := links[0]
		for _, l := range links {
			if l.CreatedAt.After(latest.CreatedAt) {
				latest = l
			}
		}
		if s.cache != nil {
			if err := s.cache.SetLink(ctx, latest.ShortURL, latest); err != nil {
				log.Ctx(ctx).Error("ошибка сохранения в кэш", "error", err)
			}
		}
		log.Ctx(ctx).Info("найдена существующая ссылка", "short_url", latest.ShortURL, "original_url", originalURL)

		return toResponseLink(latest), nil
	}

	// 3. Генерируем shortURL, если не задан
	shortURL := customUrl
	if shortURL == "" {
		for {
			shortURL = NewRandomString(0)
			existing, err := s.link.GetLinkByShortURL(ctx, shortURL)
			if err != nil {
				return nil, err // ошибка БД
			}
			if existing == nil {
				break
			}
		}
	}

	// 4. Создаём новую ссылку
	link, err := s.link.CreateLink(ctx, originalURL, shortURL, customUrl != "")
	if err != nil {
		return nil, err
	}

	// 5. Сохраняем в кэш
	if s.cache != nil {
		if err := s.cache.SetLink(ctx, shortURL, link); err != nil {
			log.Ctx(ctx).Error("ошибка сохранения в кэш", "error", err)
		}
	}

	log.Ctx(ctx).Info("новая короткая ссылка создана",
		"short_url", shortURL,
		"original_url", originalURL,
		"is_custom", customUrl != "")

	return toResponseLink(link), nil
}

// ShortLinkInfo возвращает информацию о ссылке по shortURL
func (s *Service) ShortLinkInfo(ctx context.Context, log logger.Logger, shortURL string) (*ResponseLink, error) {

	if s.cache != nil {
		link, err := s.cache.GetLink(ctx, shortURL)
		if err != nil {
			log.Ctx(ctx).Error("ошибка получения из кэша", "error", err)
		}
		if link != nil {
			log.Ctx(ctx).Debug("ссылка получена из кэша", "short_url", shortURL)
			return toResponseLink(link), nil
		}
	}

	link, err := s.link.GetLinkByShortURL(ctx, shortURL)
	if err != nil {
		return nil, err
	}
	if link == nil {
		log.Ctx(ctx).Info("ссылка не найдена в БД", "short_url", shortURL)
		return nil, nil
	}

	if s.cache != nil {
		if err := s.cache.SetLink(ctx, shortURL, link); err != nil {
			log.Ctx(ctx).Error("ошибка сохранения в кэш", "error", err)
		}
	}

	log.Ctx(ctx).Debug("ссылка получена из БД", "short_url", shortURL)

	return toResponseLink(link), nil
}

// ShortLinkAnalytics возвращает аналитику по ссылке: список переходов и агрегированные данные
// (агрегация на стороне БД за последний месяц (для дней и месяцев) и за всё время (по User-Agent)
func (s *Service) ShortLinkAnalytics(ctx context.Context, log logger.Logger, shortURL string) (*ResponseAnalytics, error) {

	link, err := s.link.GetLinkByShortURL(ctx, shortURL)
	if err != nil {
		return nil, err
	}
	if link == nil {
		log.Ctx(ctx).Info("ссылка не найдена при запросе аналитики", "short_url", shortURL)
		return nil, nil
	}

	// получаем все переходы
	analytics, err := s.analytics.GetAnalyticsByLinkID(ctx, link.ID)
	if err != nil {
		return nil, err
	}

	// получаем агрегаты
	from := time.Now().AddDate(0, -1, 0) // последний месяц для демонстрации
	to := time.Now()

	clicksByDay, err := s.analytics.CountClicksByDay(ctx, link.ID, from, to)
	if err != nil {
		log.Ctx(ctx).Error("ошибка агрегации по дням", "error", err)
		// не фатально, можно оставить пустым
	}

	clicksByMonth, err := s.analytics.CountClicksByMonth(ctx, link.ID, from, to)
	if err != nil {
		log.Ctx(ctx).Error("ошибка агрегации по месяцам", "error", err)
		// не фатально, можно оставить пустым
	}

	clicksByUA, err := s.analytics.CountClicksByUserAgent(ctx, link.ID)
	if err != nil {
		log.Ctx(ctx).Error("ошибка агрегации по user-agent", "error", err)
		// не фатально, можно оставить пустым
	}

	// преобразуем записи в формат ответа
	followLinks := make([]FollowLink, len(analytics))
	for i, a := range analytics {
		followLinks[i] = FollowLink{
			AccessedAt: a.AccessedAt,
			UserAgent:  a.UserAgent,
			IPAddress:  a.IPAddress.String(),
			Referer:    a.Referer,
		}
	}

	log.Ctx(ctx).Info("аналитика по ссылке получена", "short_url", shortURL, "clicks_count", len(analytics))

	return &ResponseAnalytics{
		Link:              *toResponseLink(link),
		Analytics:         followLinks,
		ClicksByDay:       clicksByDay,
		ClicksByMonth:     clicksByMonth,
		ClicksByUserAgent: clicksByUA,
	}, nil
}

// LastLinks возвращает последние ссылки (по умолчанию 20 строк)
func (s *Service) LastLinks(ctx context.Context, log logger.Logger) ([]*ResponseLink, error) {

	links, err := s.link.GetLinks(ctx)
	if err != nil {
		log.Ctx(ctx).Error("ошибка получения последних ссылок", "error", err)
		return nil, err
	}

	result := make([]*ResponseLink, len(links))
	for i, l := range links {
		result[i] = toResponseLink(l)
	}

	log.Ctx(ctx).Info("последние ссылки запрошены", "count", len(result))

	return result, nil
}

// RecordClick сохраняет информацию о переходе поссылке
func (s *Service) RecordClick(ctx context.Context, log logger.Logger, linkID int, userAgent, ip, referer string) error {

	err := s.analytics.SaveAnalytics(ctx, linkID, time.Now(), userAgent, ip, referer)
	if err != nil {
		log.Ctx(ctx).Error("ошибка сохранения аналитики", "error", err, "link_id", linkID)
		return err
	}

	log.Ctx(ctx).Debug("переход сохранён", "link_id", linkID, "user_agent", userAgent)

	return nil
}

// IncrementClicks увеличивает счётчик переходов по ссылке на единицу
func (s *Service) IncrementClicks(ctx context.Context, log logger.Logger, linkID int64) error {

	err := s.link.IncrementClicks(ctx, linkID)
	if err != nil {
		log.Ctx(ctx).Error("ошибка увеличения счётчика", "error", err, "link_id", linkID)
		return err
	}

	return nil
}

// SearchByOriginalURL ищет ссылки, OriginalURL которых содержит query (регистронезависимо)
func (s *Service) SearchByOriginalURL(ctx context.Context, log logger.Logger, query string) ([]*ResponseLink, error) {

	links, err := s.link.SearchByOriginalURL(ctx, query)
	if err != nil {
		log.Ctx(ctx).Error("ошибка поиска по OriginalURL", "error", err, "query", query)
		return nil, err
	}

	result := make([]*ResponseLink, len(links))
	for i, l := range links {
		result[i] = toResponseLink(l)
	}

	log.Ctx(ctx).Info("поиск по OriginalURL выполнен", "query", query, "found", len(result))

	return result, nil
}

// SearchByShortURL ищет ссылки, ShortURL которых содержит query (регистронезависимо)
func (s *Service) SearchByShortURL(ctx context.Context, log logger.Logger, query string) ([]*ResponseLink, error) {

	links, err := s.link.SearchByShortURL(ctx, query)
	if err != nil {
		log.Ctx(ctx).Error("ошибка поиска по ShortURL", "error", err, "query", query)
		return nil, err
	}

	result := make([]*ResponseLink, len(links))
	for i, l := range links {
		result[i] = toResponseLink(l)
	}

	log.Ctx(ctx).Info("поиск по ShortURL выполнен", "query", query, "found", len(result))

	return result, nil
}

// toResponseLink преобразует db.Link в service.ResponseLink
func toResponseLink(l *db.Link) *ResponseLink {

	return &ResponseLink{
		ID:          l.ID,
		ShortURL:    l.ShortURL,
		OriginalURL: l.OriginalURL,
		CreatedAt:   l.CreatedAt,
		ClicksCount: l.ClicksCount,
	}
}
