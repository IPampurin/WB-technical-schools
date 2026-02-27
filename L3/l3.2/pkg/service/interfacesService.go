package service

import (
	"context"

	"github.com/wb-go/wbf/logger"
)

type ServiceMethods interface {
	// CreateShortLink создаёт новую короткую ссылку
	CreateShortLink(ctx context.Context, log logger.Logger, originalURL, customShort string) (*ResponseLink, error)

	// ShortLinkInfo возвращает информацию о ссылке по её короткому идентификатору (для редиректа)
	ShortLinkInfo(ctx context.Context, log logger.Logger, shortURL string) (*ResponseLink, error)

	// ShortLinkAnalytics возвращает детальную информацию о ссылке и все переходы по ней
	ShortLinkAnalytics(ctx context.Context, log logger.Logger, shortURL string) (*ResponseAnalytics, error)

	// LastLinks возвращает список последних сокращённых ссылок
	LastLinks(ctx context.Context, log logger.Logger) ([]*ResponseLink, error)

	// RecordClick сохраняет информацию о переходе по ссылке (после редиректа)
	RecordClick(ctx context.Context, log logger.Logger, linkID int, userAgent, ip, referer string) error

	// IncrementClicks увеличивает счётчик переходов по ссылке (вызывается вместе с RecordClick)
	IncrementClicks(ctx context.Context, log logger.Logger, linkID int64) error

	// SearchByOriginalURL ищет ссылки, OriginalURL которых содержит подстроку query
	SearchByOriginalURL(ctx context.Context, log logger.Logger, query string) ([]*ResponseLink, error)

	// SearchByShortURL ищет ссылки, ShortURL которых содержит подстроку query
	SearchByShortURL(ctx context.Context, log logger.Logger, query string) ([]*ResponseLink, error)
}
