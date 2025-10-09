package model

import (
	"time"
)

type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,min=10,max=50"`
	TrackNumber       string    `json:"track_number" validate:"required,min=5,max=20"`
	Entry             string    `json:"entry" validate:"required"`
	Delivery          Delivery  `json:"delivery" validate:"required"`
	Payment           Payment   `json:"payment" validate:"required"`
	Items             []Item    `json:"items" validate:"required,min=1,max=100,dive"`
	Locale            string    `json:"locale" validate:"required,len=2"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id" validate:"required"`
	DeliveryService   string    `json:"delivery_service" validate:"required"`
	ShardKey          string    `json:"shardkey" validate:"required"`
	SmID              int       `json:"sm_id" validate:"required,min=1"`
	DateCreated       time.Time `json:"date_created" validate:"required"`
	OofShard          string    `json:"oof_shard" validate:"required"`
}

type Delivery struct {
	Name    string `json:"name" validate:"required,min=2,max=100"`
	Phone   string `json:"phone" validate:"required,min=10,max=20"`
	Zip     string `json:"zip" validate:"required,min=3,max=10"`
	City    string `json:"city" validate:"required,min=2,max=50"`
	Address string `json:"address" validate:"required,min=5,max=200"`
	Region  string `json:"region" validate:"required,min=2,max=50"`
	Email   string `json:"email" validate:"required,email"`
}

type Payment struct {
	Transaction  string `json:"transaction" validate:"required"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency" validate:"required,len=3"`
	Provider     string `json:"provider" validate:"required"`
	Amount       int    `json:"amount" validate:"required,min=1,max=1000000"`
	PaymentDT    int    `json:"payment_dt" validate:"required,min=1"`
	Bank         string `json:"bank" validate:"required"`
	DeliveryCost int    `json:"delivery_cost" validate:"min=0,max=100000"`
	GoodsTotal   int    `json:"goods_total" validate:"required,min=1,max=1000000"`
	CustomFee    int    `json:"custom_fee" validate:"min=0,max=100000"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id" validate:"required,min=1"`
	TrackNumber string `json:"track_number" validate:"required,min=5,max=20"`
	Price       int    `json:"price" validate:"required,min=1,max=100000"`
	Rid         string `json:"rid" validate:"required"`
	Name        string `json:"name" validate:"required,min=1,max=200"`
	Sale        int    `json:"sale" validate:"min=0,max=100"`
	Size        string `json:"size" validate:"required"`
	TotalPrice  int    `json:"total_price" validate:"required,min=1,max=100000"`
	NmID        int    `json:"nm_id" validate:"required,min=1"`
	Brand       string `json:"brand" validate:"required,min=1,max=100"`
	Status      int    `json:"status" validate:"required,min=0"`
}
