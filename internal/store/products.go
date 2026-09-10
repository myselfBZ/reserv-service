package store

import (
	"context"
	"database/sql"
)

type Product struct {
	Id 	  string `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	StockQuantity int     `json:"stock_quantity"`
}

type ProductStore struct {
	db *sql.DB
}

func (s *ProductStore) Create(ctx context.Context, p *Product) error {
	q := `INSERT INTO products(name, price, stock_quantity) VALUES($1, $2, $3) RETURNING id`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx,
		q,
		p.Name,
		p.Price,
		p.StockQuantity,
	).Scan(&p.Id)

	return err
}

func (s *ProductStore) GetById(ctx context.Context, id string) (*Product, error) {
	q := `SELECT id, name, price, stock_quantity FROM products WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	var p Product
	err := s.db.QueryRowContext(
		ctx,
		q,
		id,
	).Scan(
		&p.Id,
		&p.Name,
		&p.Price,
		&p.StockQuantity,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrResourceNotFound
		default:
			return nil, err
		}
	}

	return &p, nil
}
