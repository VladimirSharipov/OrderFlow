package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"wbtest/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB подключение к БД
type DB struct {
	pool *pgxpool.Pool
	// DB экспортированное поле для доступа к подключению (для миграций)
	DB *pgxpool.Pool
}

// New создает подключение к БД
func New(connStr string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	return &DB{pool: pool, DB: pool}, nil
}

// Close закрывает подключение
func (db *DB) Close() {
	db.pool.Close()
}

// LoadAllOrders загружает все заказы
func (db *DB) LoadAllOrders(ctx context.Context) ([]*model.Order, error) {
	query := `
	SELECT 
	  o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, 
	  o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created::text, o.oof_shard,
	  row_to_json(d.*),
	  row_to_json(p.*),
	  COALESCE(json_agg(i.*) FILTER (WHERE i.id IS NOT NULL), '[]')
	FROM orders o
	JOIN delivery d ON d.order_uid = o.order_uid
	JOIN payment p ON p.order_uid = o.order_uid
	LEFT JOIN items i ON i.order_uid = o.order_uid
	GROUP BY o.order_uid, d.*, p.*
	`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var o model.Order
		var dateCreated time.Time
		var deliveryJSON, paymentJSON []byte
		var itemsJSON []byte

		err := rows.Scan(
			&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
			&o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.SmID, &dateCreated, &o.OofShard,
			&deliveryJSON, &paymentJSON, &itemsJSON,
		)
		if err != nil {
			return nil, err
		}

		o.DateCreated = dateCreated

		// Парсим JSON данные для связанных сущностей
		if err := json.Unmarshal(deliveryJSON, &o.Delivery); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(paymentJSON, &o.Payment); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(itemsJSON, &o.Items); err != nil {
			return nil, err
		}

		orders = append(orders, &o)
	}

	// На всякий случай проверим ошибку итератора
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

// GetOrderByUID загружает заказ по UID
func (db *DB) GetOrderByUID(ctx context.Context, orderUID string) (*model.Order, error) {
	query := `
	SELECT 
	  o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, 
	  o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created::text, o.oof_shard,
	  row_to_json(d.*),
	  row_to_json(p.*),
	  COALESCE(json_agg(i.*) FILTER (WHERE i.id IS NOT NULL), '[]')
	FROM orders o
	JOIN delivery d ON d.order_uid = o.order_uid
	JOIN payment p ON p.order_uid = o.order_uid
	LEFT JOIN items i ON i.order_uid = o.order_uid
	WHERE o.order_uid = $1
	GROUP BY o.order_uid, d.*, p.*
	`

	var order model.Order
	var deliveryJSON, paymentJSON []byte
	var itemsJSON []byte
	var dateCreatedStr string

	err := db.pool.QueryRow(ctx, query, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerID, &order.DeliveryService, &order.ShardKey, &order.SmID, &dateCreatedStr, &order.OofShard,
		&deliveryJSON, &paymentJSON, &itemsJSON,
	)
	if err != nil {
		return nil, err
	}

	// Парсим дату создания
	if dateCreated, err := time.Parse(time.RFC3339, dateCreatedStr); err == nil {
		order.DateCreated = dateCreated
	}

	// Парсим JSON поля
	if err := json.Unmarshal(deliveryJSON, &order.Delivery); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(paymentJSON, &order.Payment); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, err
	}

	return &order, nil
}

// SaveOrder сохраняет заказ в БД
func (db *DB) SaveOrder(ctx context.Context, order *model.Order) error {
	// Небольшие проверки входных данных чтобы не писать мусор
	if order == nil {
		return errors.New("order is nil")
	}
	if order.OrderUID == "" {
		return errors.New("order uid is empty")
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	// Сохраняем основную информацию о заказе
	_, err = tx.Exec(ctx, `
		INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard) 
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) 
		ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.ShardKey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return err
	}

	// Сохраняем информацию о доставке
	_, err = tx.Exec(ctx, `
		INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email) 
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) 
		ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return err
	}

	// Сохраняем информацию об оплате
	_, err = tx.Exec(ctx, `
		INSERT INTO payment (transaction, order_uid, request_id, currency, provider, 
			amount, payment_dt, bank, delivery_cost, goods_total, custom_fee) 
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) 
		ON CONFLICT (transaction) DO NOTHING`,
		order.Payment.Transaction, order.OrderUID, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return err
	}

	// Сохраняем товары заказа
	for _, item := range order.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, 
				sale, size, total_price, nm_id, brand, status) 
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid,
			item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return err
		}
	}

	return nil
}
