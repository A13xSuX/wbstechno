package database

import (
	"time"
)

type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,alphanumdash"`    // ИЗМЕНЕНО
	TrackNumber       string    `json:"track_number" validate:"required,alphanumdash"` // ИЗМЕНЕНО
	Entry             string    `json:"entry" validate:"required,alpha"`
	Delivery          Delivery  `json:"delivery" validate:"required"`
	Payment           Payment   `json:"payment" validate:"required"`
	Items             []Item    `json:"items" validate:"required,min=1,dive"`
	Locale            string    `json:"locale" validate:"required,alpha"`
	InternalSignature string    `json:"internal_signature" validate:"omitempty"`
	CustomerID        string    `json:"customer_id" validate:"required,alphanumdash"` // ИЗМЕНЕНО
	DeliveryService   string    `json:"delivery_service" validate:"required,alpha"`
	Shardkey          string    `json:"shardkey" validate:"required,alphanumdash"` // ИЗМЕНЕНО
	SmID              int       `json:"sm_id" validate:"required,min=0"`
	DateCreated       time.Time `json:"date_created" validate:"required"`
	OofShard          string    `json:"oof_shard" validate:"required,alphanumdash"` // ИЗМЕНЕНО
}

type Delivery struct {
	Name    string `json:"name" validate:"required,min=2"`
	Phone   string `json:"phone" validate:"required,e164"` // E.164 format: +1234567890
	Zip     string `json:"zip" validate:"required,numeric"`
	City    string `json:"city" validate:"required,alphaunicode"`
	Address string `json:"address" validate:"required,min=5"`
	Region  string `json:"region" validate:"required,alphaunicode"`
	Email   string `json:"email" validate:"required,email"`
}

type Payment struct {
	Transaction  string `json:"transaction" validate:"required,uuid"` // UUID format
	RequestID    string `json:"request_id" validate:"omitempty"`
	Currency     string `json:"currency" validate:"required,alpha,uppercase,min=3,max=3"`
	Provider     string `json:"provider" validate:"required,alpha"`
	Amount       int    `json:"amount" validate:"required,min=1"`
	PaymentDt    int64  `json:"payment_dt" validate:"required,min=0"`
	Bank         string `json:"bank" validate:"required,alpha"`
	DeliveryCost int    `json:"delivery_cost" validate:"min=0"`
	GoodsTotal   int    `json:"goods_total" validate:"min=0"`
	CustomFee    int    `json:"custom_fee" validate:"min=0"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id" validate:"required,min=0"`
	TrackNumber string `json:"track_number" validate:"required,alphanum"`
	Price       int    `json:"price" validate:"required,min=1"`
	Rid         string `json:"rid" validate:"required,alphanum"`
	Name        string `json:"name" validate:"required,min=2"`
	Sale        int    `json:"sale" validate:"min=0"`
	Size        string `json:"size" validate:"omitempty"`
	TotalPrice  int    `json:"total_price" validate:"min=0"`
	NmID        int    `json:"nm_id" validate:"min=0"`
	Brand       string `json:"brand" validate:"required,min=2"`
	Status      int    `json:"status" validate:"min=0"`
}
