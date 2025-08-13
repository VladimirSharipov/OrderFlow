package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"wbtest/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB это подключение к базе данных
type DB struct {
	pool *pgxpool.Pool
}

// New создает новое подключение к базе данных
func New(connStr string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

// Close закрывает подключение к базе данных
func (db *DB) Close() {
	db.pool.Close()
}

// LoadAllOrders загружает все заказы из базы данных
// используем один SQL запрос чтобы загрузить все данные сразу
func (db *DB) LoadAllOrders(ctx context.Context) ([]*model.Order, error) {
	query := `
	SELECT 
	  o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, 
	  o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
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

		// парсим JSON данные для доставки оплаты и товаров
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

// SaveOrder сохраняет заказ в базу данных
// используем транзакцию чтобы все данные сохранились или ничего
func (db *DB) SaveOrder(ctx context.Context, order *model.Order) error {
	// проверяем что заказ не пустой
	if order == nil {
		return errors.New("заказ пустой")
	}
	if order.OrderUID == "" {
		return errors.New("ID заказа пустой")
	}

	// начинаем транзакцию
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

	// сохраняем основную информацию о заказе
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

	// сохраняем информацию о доставке
	_, err = tx.Exec(ctx, `
		INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email) 
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) 
		ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return err
	}

	// сохраняем информацию об оплате
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

	// сохраняем товары заказа
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
