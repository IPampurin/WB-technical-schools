package models

import (
	"time"
)

// Заказ
type Order struct {
	ID                uint `gorm:"primaryKey"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	OrderUID          string    `json:"order_uid" validate:"required"`
	TrackNumber       string    `json:"track_number" validate:"required"`
	Entry             string    `json:"entry" validate:"required"`
	Delivery          Delivery  `json:"delivery" gorm:"foreignKey:OrderID" validate:"required"`
	Payment           Payment   `json:"payment" gorm:"foreignKey:OrderID" validate:"required"`
	Items             []Item    `json:"items" gorm:"foreignKey:OrderID" validate:"required,min=1,dive"`
	Locale            string    `json:"locale" validate:"required"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id" validate:"required"`
	DeliveryService   string    `json:"delivery_service" validate:"required"`
	Shardkey          string    `json:"shardkey" validate:"required"`
	SMID              int       `json:"sm_id" validate:"required,min=1"`
	DateCreated       time.Time `json:"date_created"`
	OOFShard          string    `json:"oof_shard"`
}

// Доставка
type Delivery struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	OrderID   uint   `json:"-"`
	Name      string `json:"name" validate:"required"`
	Phone     string `json:"phone" validate:"required"`
	Zip       string `json:"zip" validate:"required"`
	City      string `json:"city" validate:"required"`
	Address   string `json:"address" validate:"required"`
	Region    string `json:"region" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
}

// Оплата
type Payment struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	OrderID      uint    `json:"-"`
	Transaction  string  `json:"transaction" validate:"required"`
	RequestID    string  `json:"request_id"`
	Currency     string  `json:"currency" validate:"required"`
	Provider     string  `json:"provider" validate:"required"`
	Amount       float64 `json:"amount" validate:"required,min=0"`
	PaymentDT    int64   `json:"payment_dt" validate:"required,min=1"`
	Bank         string  `json:"bank" validate:"required"`
	DeliveryCost float64 `json:"delivery_cost" validate:"min=0"`
	GoodsTotal   float64 `json:"goods_total" validate:"min=0"`
	CustomFee    float64 `json:"custom_fee" validate:"min=0"`
}

// Позиция заказа
type Item struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	OrderID     uint    `json:"-"`
	ChrtID      int     `json:"chrt_id" validate:"required,min=1"`
	TrackNumber string  `json:"track_number"`
	Price       float64 `json:"price" validate:"required,min=0"`
	RID         string  `json:"rid" validate:"required"`
	Name        string  `json:"name" validate:"required"`
	Sale        float64 `json:"sale" validate:"min=0,max=100"`
	Size        string  `json:"size" validate:"required"`
	TotalPrice  float64 `json:"total_price" validate:"required,min=0"`
	NMID        int     `json:"nm_id" validate:"required,min=1"`
	Brand       string  `json:"brand" validate:"required"`
	Status      int     `json:"status" validate:"required,min=0"`
}
