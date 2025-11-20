package models

import (
	"time"

	"gorm.io/gorm"
)

// Заказ
type Order struct {
	gorm.Model
	OrderUID          string    `json:"order_uid" gorm:"unique_index;not null" validate:"required"`
	TrackNumber       string    `json:"track_number" gorm:"index" validate:"required"`
	Entry             string    `json:"entry" validate:"required"`
	Delivery          Delivery  `json:"delivery" gorm:"foreignKey:OrderID" validate:"required"`
	Payment           Payment   `json:"payment" gorm:"foreignKey:OrderID" validate:"required"`
	Items             []Item    `json:"items" gorm:"foreignKey:OrderID" validate:"required,min=1,dive"`
	Locale            string    `json:"locale" validate:"required"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id" validate:"required"`
	DeliveryService   string    `json:"delivery_service" validate:"required"`
	Shardkey          string    `json:"shardkey" validate:"required"`
	SMID              int       `json:"sm_id" gorm:"type:smallint" validate:"required,min=1"`
	DateCreated       time.Time `json:"date_created" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	OOFShard          string    `json:"oof_shard"`
}

// Доставка
type Delivery struct {
	gorm.Model
	OrderID uint   `json:"-"`
	Name    string `json:"name" gorm:"size:100" validate:"required"`
	Phone   string `json:"phone" gorm:"size:20" validate:"required"`
	Zip     string `json:"zip" gorm:"size:20" validate:"required"`
	City    string `json:"city" gorm:"size:100" validate:"required"`
	Address string `json:"address" gorm:"size:255" validate:"required"`
	Region  string `json:"region" gorm:"size:100" validate:"required"`
	Email   string `json:"email" gorm:"unique_index;type:varchar(100)" validate:"required,email"`
}

// Оплата
type Payment struct {
	gorm.Model
	OrderID      uint    `json:"-"`
	Transaction  string  `json:"transaction" gorm:"unique_index" validate:"required"`
	RequestID    string  `json:"request_id"`
	Currency     string  `json:"currency" gorm:"size:3" validate:"required"`
	Provider     string  `json:"provider" gorm:"size:50" validate:"required"`
	Amount       float64 `json:"amount" gorm:"type:decimal(10,2)" validate:"required,min=0"`
	PaymentDT    int64   `json:"payment_dt" gorm:"type:bigint" validate:"required,min=1"`
	Bank         string  `json:"bank" gorm:"size:50" validate:"required"`
	DeliveryCost float64 `json:"delivery_cost" gorm:"type:decimal(10,2)" validate:"min=0"`
	GoodsTotal   float64 `json:"goods_total" gorm:"type:decimal(10,2)" validate:"min=0"`
	CustomFee    float64 `json:"custom_fee" gorm:"type:decimal(10,2)" validate:"min=0"`
}

// Позиция заказа
type Item struct {
	gorm.Model
	OrderID     uint    `json:"-"`
	ChrtID      int     `json:"chrt_id" gorm:"type:integer" validate:"required,min=1"`
	TrackNumber string  `json:"track_number"`
	Price       float64 `json:"price" gorm:"type:decimal(10,2)" validate:"required,min=0"`
	RID         string  `json:"rid" gorm:"size:100" validate:"required"`
	Name        string  `json:"name" gorm:"size:255" validate:"required"`
	Sale        float64 `json:"sale" gorm:"type:decimal(5,2)" validate:"min=0,max=100"`
	Size        string  `json:"size" gorm:"size:20" validate:"required"`
	TotalPrice  float64 `json:"total_price" gorm:"type:decimal(10,2)" validate:"required,min=0"`
	NMID        int     `json:"nm_id" gorm:"type:integer" validate:"required,min=1"`
	Brand       string  `json:"brand" gorm:"size:100" validate:"required"`
	Status      int     `json:"status" gorm:"type:smallint" validate:"required,min=0"`
}
