package api

import (
	"encoding/json"
	"time"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"
)

// itemToAPI преобразует доменный товар в API-ответ
func itemToAPI(item *domain.Item) *Item {

	if item == nil {
		return nil
	}

	return &Item{
		ID:       item.ID,
		Name:     item.Name,
		Quantity: item.Quantity,
		Price:    item.Price,
	}
}

// itemFromCreateRequest преобразует запрос на создание/обновление в доменную модель (ID и даты - позже)
func itemFromCreateRequest(req *CreateUpdateItemRequest) *domain.Item {

	if req == nil {
		return nil
	}

	return &domain.Item{
		Name:     req.Name,
		Quantity: req.Quantity,
		Price:    req.Price,
	}
}

// itemFromUpdateRequest аналогичен itemFromCreateRequest, но с заполнением ID
func itemFromUpdateRequest(id int, req *CreateUpdateItemRequest) *domain.Item {

	if req == nil {
		return nil
	}

	return &domain.Item{
		ID:       id,
		Name:     req.Name,
		Quantity: req.Quantity,
		Price:    req.Price,
	}
}

// historyToAPI преобразует доменную запись истории в API-ответ
func historyToAPI(h *domain.ItemHistory) *HistoryRecord {

	if h == nil {
		return nil
	}

	rec := &HistoryRecord{
		ID:        h.ID,
		ItemID:    h.ItemID,
		Action:    h.Action,
		ChangedBy: h.ChangedBy,
		ChangedAt: h.ChangedAt.Format(time.RFC3339),
	}

	if len(h.OldData) > 0 {
		var old interface{}
		if err := json.Unmarshal(h.OldData, &old); err == nil {
			rec.OldData = old
		} else {
			rec.OldData = string(h.OldData)
		}
	}

	if len(h.NewData) > 0 {
		var new interface{}
		if err := json.Unmarshal(h.NewData, &new); err == nil {
			rec.NewData = new
		} else {
			rec.NewData = string(h.NewData)
		}
	}

	return rec
}

// parseTimeSafe преобразует строку в формате RFC3339 в time.Time ("" -> zero)
func parseTimeSafe(str string) (time.Time, error) {

	if str == "" {
		return time.Time{}, nil
	}

	return time.Parse(time.RFC3339, str)
}
