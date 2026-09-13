package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

const CancelPendingInterval = time.Minute * 15

var (
	ErrOrderCannotBeCancelled  = errors.New("order can't be cancelled")
	ErrInvalidProductId        = errors.New("invalid product id: product does not exist")
	ErrInsufficientStock       = errors.New("insufficient stock quantity in the inventory")
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
)

type OrderStatus string

const (
	OrderStatusPending   = "pending"
	OrderStatusConfirmed = "confirmed"
	OrderStatusCancelled = "cancelled"
)

type OrderItem struct {
	OrderId   string  `json:"order_id"`
	ProductId string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type Order struct {
	Id             string      `json:"id"`
	UserId         string      `json:"user_id"`
	IdempotencyKey string      `json:"idempotency_key"`
	TotalPrice     float64     `json:"total_price"`
	Status         OrderStatus `json:"status"`
	PlacedAt       time.Time   `json:"placed_at"`

	OrderItems []OrderItem `json:"order_items"`
}

type OrderStore struct {
	db *sql.DB
}


func (s *OrderStore) GetByIdempotencyKey(ctx context.Context, userId, key string) (*Order, error) {
	q := `SELECT 
			id, 
			user_id, 
			total_price, 
			status, 
			idempotency_key,
			placed_at 
		FROM orders WHERE user_id = $1 AND idempotency_key = $2`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var o Order
	err := s.db.QueryRowContext(ctx, q, userId, key).Scan(
		&o.Id,
		&o.UserId,
		&o.TotalPrice,
		&o.Status,
		&o.IdempotencyKey,
		&o.PlacedAt,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrResourceNotFound
		default:
			return nil, err
		}
	}
	r, err := s.db.QueryContext(ctx, `SELECT * FROM order_items WHERE order_id = $1`, o.Id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}
	items := []OrderItem{}
	for r.Next() {
		var it OrderItem
		err := r.Scan(
			&it.OrderId,
			&it.ProductId,
			&it.Quantity,
			&it.UnitPrice,
		)

		if err != nil {
			return nil, err
		}

		items = append(items, it)
	}
	o.OrderItems = items
	return &o, nil
}

func (s *OrderStore) GetById(ctx context.Context, id string) (*Order, error) {
	q := `SELECT 
			id, user_id, total_price, status, idempotency_key ,placed_at 
		FROM orders WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var o Order
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&o.Id,
		&o.UserId,
		&o.TotalPrice,
		&o.Status,
		&o.IdempotencyKey,
		&o.PlacedAt,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrResourceNotFound
		default:
			return nil, err
		}
	}

	items := []OrderItem{}
	r, err := s.db.QueryContext(ctx, `SELECT * FROM order_items WHERE order_id = $1`, o.Id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}
	for r.Next() {
		var it OrderItem
		err := r.Scan(
			&it.OrderId,
			&it.ProductId,
			&it.Quantity,
			&it.UnitPrice,
		)

		if err != nil {
			return nil, err
		}

		items = append(items, it)
	}
	o.OrderItems = items
	return &o, nil
}

func (s *OrderStore) Create(ctx context.Context, o *Order) (err error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var calculatedTotal float64
	for i := range o.OrderItems {
		var stock int
		var unitPrice float64

		err := tx.QueryRowContext(
			ctx,
			`SELECT stock_quantity, price FROM products WHERE id = $1 FOR UPDATE`,
			o.OrderItems[i].ProductId,
		).Scan(&stock, &unitPrice)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidProductId
			}
			return err
		}

		if stock < o.OrderItems[i].Quantity {
			return ErrInsufficientStock
		}

		o.OrderItems[i].UnitPrice = unitPrice
		calculatedTotal += unitPrice * float64(o.OrderItems[i].Quantity)
	}

	o.TotalPrice = calculatedTotal

	q := `INSERT INTO orders(user_id, total_price, idempotency_key) VALUES($1, $2, $3) RETURNING id, placed_at, status`

	err = tx.QueryRowContext(
		ctx,
		q,
		o.UserId,
		o.TotalPrice,
		o.IdempotencyKey,
	).Scan(
		&o.Id,
		&o.PlacedAt,
		&o.Status,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "uq_orders_user_idempotency" {
			return ErrDuplicateIdempotencyKey
		}
		return err
	}

	for _, it := range o.OrderItems {
		err = s.createOrderItem(ctx, tx, o.Id, &it)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(
			ctx,
			`UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2`,
			it.Quantity,
			it.ProductId,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *OrderStore) Confirm(ctx context.Context, orderId string) error {
	q := `UPDATE order SET status = 'confirmed' WHERE id = $1 AND status NOT IN ('cancelled', 'confirmed')`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	r, err := s.db.ExecContext(ctx, q, orderId)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (s *OrderStore) createOrderItem(ctx context.Context, tx *sql.Tx, orderId string, it *OrderItem) error {
	query := `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price)
		VALUES ($1, $2, $3, $4)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		orderId,
		it.ProductId,
		it.Quantity,
		it.UnitPrice,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return ErrInvalidProductId
		}
		return err
	}
	it.OrderId = orderId
	return nil
}

func (s *OrderStore) Cancel(ctx context.Context, orderId string) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r, err := tx.QueryContext(ctx, `SELECT * FROM order_items WHERE order_id = $1`, orderId)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return ErrResourceNotFound
		}
		return err
	}
	items := []OrderItem{}
	for r.Next() {
		var it OrderItem
		err := r.Scan(
			&it.OrderId,
			&it.ProductId,
			&it.Quantity,
			&it.UnitPrice,
		)

		if err != nil {
			return err
		}

		items = append(items, it)
	}
	for _, it := range items {
		_, err = tx.ExecContext(
			ctx, 
			`UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, 
			it.Quantity, 
			it.ProductId,
		)

		if err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(
		ctx, 
		`UPDATE orders SET status = 'cancelled' WHERE id = $1 AND status NOT IN ('cancelled', 'confirmed')`, 
		orderId,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrOrderCannotBeCancelled
	}
	return tx.Commit()
}

func (s *OrderStore) CancelStale(ctx context.Context) ([]string, error) {
	q := `SELECT 
				o.id,
				o.user_id,
				o.status,
				o.total_price,
				o.placed_at,
				it.product_id,
				it.quantity
		FROM orders o 
		JOIN order_items it ON o.id = it.order_id 
		WHERE o.id IN (
			SELECT id FROM orders 
			WHERE status = 'pending' AND placed_at < NOW() - $1::interval
			ORDER BY placed_at ASC
			LIMIT $2 
			FOR UPDATE SKIP LOCKED
	)`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, q, CancelPendingInterval.String(), 50)
	if err != nil {
		return nil, err
	}
	var orderedIDs []string
	orderMap := make(map[string]*Order)
	for rows.Next() {
		var o Order
		var(
			productId string
			quantity int
		)
		err := rows.Scan(
			&o.Id,
			&o.UserId,
			&o.Status,
			&o.TotalPrice,
			&o.PlacedAt,
			&productId,
			&quantity,
		)

		if err != nil {
			return nil, err
		}

		ord, exists := orderMap[o.Id]
		if !exists {
			ord = &o
			orderMap[o.Id] = ord
			orderedIDs = append(orderedIDs, o.Id)
		}

		ord.OrderItems = append(ord.OrderItems, OrderItem{
			ProductId: productId,
			Quantity: quantity,
		})
	}

	if len(orderedIDs) == 0 {
		return orderedIDs, nil
	}

	for _, v := range orderMap {

		for _, it := range v.OrderItems {
			_, err = tx.ExecContext(
				ctx, 
				`UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, 
				it.Quantity, 
				it.ProductId,
			)

			if err != nil {
				return nil, err
			}
		}

		res, err := tx.ExecContext(
			ctx, 
			`UPDATE orders SET status = 'cancelled' WHERE id = $1 AND status NOT IN ('cancelled', 'confirmed')`, 
			v.Id,
		)
		if err != nil {
			return nil, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			return nil, ErrOrderCannotBeCancelled
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderedIDs, nil
}
