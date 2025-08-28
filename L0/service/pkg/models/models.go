package models

import (
	"time"

	"gorm.io/gorm"
)

// Заказ
type Order struct {
	gorm.Model
	OrderUID          string    `json:"order_uid" gorm:"unique_index;not null"`
	TrackNumber       string    `json:"track_number" gorm:"index"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery" gorm:"foreignKey:OrderID"`
	Payment           Payment   `json:"payment" gorm:"foreignKey:OrderID"`
	Items             []Item    `json:"items" gorm:"foreignKey:OrderID"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	Shardkey          string    `json:"shardkey"`
	SMID              int       `json:"sm_id" gorm:"type:smallint"`
	DateCreated       time.Time `json:"date_created" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	OOFShard          string    `json:"oof_shard"`
}

// Доставка
type Delivery struct {
	gorm.Model
	OrderID uint   `json:"-"`
	Name    string `json:"name" gorm:"size:100"`
	Phone   string `json:"phone" gorm:"size:20"`
	Zip     string `json:"zip" gorm:"size:20"`
	City    string `json:"city" gorm:"size:100"`
	Address string `json:"address" gorm:"size:255"`
	Region  string `json:"region" gorm:"size:100"`
	Email   string `json:"email" gorm:"unique_index;type:varchar(100)"`
}

// Оплата
type Payment struct {
	gorm.Model
	OrderID      uint    `json:"-"`
	Transaction  string  `json:"transaction" gorm:"unique_index"`
	RequestID    string  `json:"request_id"`
	Currency     string  `json:"currency" gorm:"size:3"`
	Provider     string  `json:"provider" gorm:"size:50"`
	Amount       float64 `json:"amount" gorm:"type:decimal(10,2)"`
	PaymentDT    int64   `json:"payment_dt" gorm:"type:bigint"`
	Bank         string  `json:"bank" gorm:"size:50"`
	DeliveryCost float64 `json:"delivery_cost" gorm:"type:decimal(10,2)"`
	GoodsTotal   float64 `json:"goods_total" gorm:"type:decimal(10,2)"`
	CustomFee    float64 `json:"custom_fee" gorm:"type:decimal(10,2)"`
}

// Позиция заказа
type Item struct {
	gorm.Model
	OrderID     uint    `json:"-"`
	ChrtID      int     `json:"chrt_id" gorm:"type:integer"`
	TrackNumber string  `json:"track_number"`
	Price       float64 `json:"price" gorm:"type:decimal(10,2)"`
	RID         string  `json:"rid" gorm:"size:100"`
	Name        string  `json:"name" gorm:"size:255"`
	Sale        float64 `json:"sale" gorm:"type:decimal(5,2)"`
	Size        string  `json:"size" gorm:"size:20"`
	TotalPrice  float64 `json:"total_price" gorm:"type:decimal(10,2)"`
	NMID        int     `json:"nm_id" gorm:"type:integer"`
	Brand       string  `json:"brand" gorm:"size:100"`
	Status      int     `json:"status" gorm:"type:smallint"`
}
