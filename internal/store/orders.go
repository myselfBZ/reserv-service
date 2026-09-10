package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)


var(
	ErrInvalidProductId = errors.New("invalid product id: product does not exist")
	ErrInsufficientStock = errors.New("insufficient stock quantity in the inventory")
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
)

type OrderItem struct {
	OrderId   string  `json:"order_id"`
	ProductId string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type Order struct {
	Id             string    `json:"id"`
	UserId         string    `json:"user_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	TotalPrice     float64   `json:"total_price"`
	Status         string    `json:"status"`
	PlacedAt       time.Time `json:"placed_at"`

	OrderItems []OrderItem `json:"order_items"`
}

type OrderStore struct {
	db *sql.DB
}

func (s *OrderStore) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM orders WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return ErrResourceNotFound
		default:
			return err
		}
	}
	return nil
}

func (s *OrderStore) GetById(ctx context.Context, id string) (*Order, error) {
	q := `SELECT id, user_id, total_price, status, placed_at FROM orders WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var o Order
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&o.Id,
		&o.UserId,
		&o.TotalPrice,
		&o.Status,
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
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "orders_idempotency_key_key" {
			return ErrDuplicateIdempotencyKey
		}
		return err
	}

	for _, it := range o.OrderItems {
		err = s.CreateOrderItem(ctx, tx, o.Id, &it)
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

func (s *OrderStore) CreateOrderItem(ctx context.Context, tx *sql.Tx, orderId string, it *OrderItem) error {
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
	return nil
}
