package model

import (
	"time"
)

type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`              // код входа
	Delivery          Delivery  `json:"delivery"`           // информация о доставке
	Payment           Payment   `json:"payment"`            // информация об оплате
	Items             []Item    `json:"items"`              // список товаров
	Locale            string    `json:"locale"`             // локаль
	InternalSignature string    `json:"internal_signature"` // внутренняя подпись
	CustomerID        string    `json:"customer_id"`        // ID клиента
	DeliveryService   string    `json:"delivery_service"`   // сервис доставки
	ShardKey          string    `json:"shardkey"`           // ключ шарда
	SmID              int       `json:"sm_id"`              // ID магазина
	DateCreated       time.Time `json:"date_created"`       // дата создания
	OofShard          string    `json:"oof_shard"`          // шард OOF
}

type Delivery struct {
	Name    string `json:"name"`    // имя получателя
	Phone   string `json:"phone"`   // телефон
	Zip     string `json:"zip"`     // почтовый индекс
	City    string `json:"city"`    // город
	Address string `json:"address"` // адрес
	Region  string `json:"region"`  // регион
	Email   string `json:"email"`   // email
}

type Payment struct {
	Transaction  string `json:"transaction"`   // ID транзакции
	RequestID    string `json:"request_id"`    // ID запроса
	Currency     string `json:"currency"`      // валюта
	Provider     string `json:"provider"`      // провайдер
	Amount       int    `json:"amount"`        // сумма
	PaymentDT    int    `json:"payment_dt"`    // дата оплаты
	Bank         string `json:"bank"`          // банк
	DeliveryCost int    `json:"delivery_cost"` // стоимость доставки
	GoodsTotal   int    `json:"goods_total"`   // общая стоимость товаров
	CustomFee    int    `json:"custom_fee"`    // комиссия
}

type Item struct {
	ChrtID      int    `json:"chrt_id"`      // ID товара
	TrackNumber string `json:"track_number"` // номер отслеживания
	Price       int    `json:"price"`        // цена
	Rid         string `json:"rid"`          // ID возврата
	Name        string `json:"name"`         // название
	Sale        int    `json:"sale"`         // скидка
	Size        string `json:"size"`         // размер
	TotalPrice  int    `json:"total_price"`  // общая цена
	NmID        int    `json:"nm_id"`        // ID номенклатуры
	Brand       string `json:"brand"`        // бренд
	Status      int    `json:"status"`       // статус
}
