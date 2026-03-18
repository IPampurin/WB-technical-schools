package db

import "github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"

// domainUserToDBUser преобразует доменную модель User в модель User базы данных
func domainUserToDBUser(u *domain.User) *User {

	if u == nil {
		return nil
	}

	return &User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

// dbUserToDomainUser преобразует модель User базы данных в доменную модель User
func dbUserToDomainUser(u *User) *domain.User {

	if u == nil {
		return nil
	}

	return &domain.User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

// domainItemToDBItem преобразует доменную модель Item в модель Item базы данных
func domainItemToDBItem(i *domain.Item) *Item {

	if i == nil {
		return nil
	}

	return &Item{
		ID:        i.ID,
		Name:      i.Name,
		Quantity:  i.Quantity,
		Price:     i.Price,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
	}
}

// dbItemToDomainItem преобразует модель Item базы данных в доменную модель Item
func dbItemToDomainItem(i *Item) *domain.Item {

	if i == nil {
		return nil
	}

	return &domain.Item{
		ID:        i.ID,
		Name:      i.Name,
		Quantity:  i.Quantity,
		Price:     i.Price,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
	}
}

// dbItemHistoryToDomainItemHistory преобразует модель ItemHistory базы данных в доменную модель ItemHistory
func dbItemHistoryToDomainItemHistory(h *ItemHistory) *domain.ItemHistory {

	if h == nil {
		return nil
	}

	return &domain.ItemHistory{
		ID:        h.ID,
		ItemID:    h.ItemID,
		Action:    h.Action,
		OldData:   h.OldData,
		NewData:   h.NewData,
		ChangedBy: h.ChangedBy,
		ChangedAt: h.ChangedAt,
	}
}

// domainHistoryFilterToDBHistoryFilter преобразует доменную модель HistoryFilter в модель HistoryFilter базы данных
func domainHistoryFilterToDBHistoryFilter(f *domain.HistoryFilter) *HistoryFilter {

	return &HistoryFilter{
		ItemID:   f.ItemID,
		FromDate: f.FromDate,
		ToDate:   f.ToDate,
		UserID:   f.UserID,
		Action:   f.Action,
		Limit:    f.Limit,
		Offset:   f.Offset,
	}
}
