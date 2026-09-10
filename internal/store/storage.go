package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrResourceNotFound = errors.New("resource not found")
)
var QueryTimeoutDuration = time.Second * 5

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		Orders:   &OrderStore{db: db},
		Products: &ProductStore{db: db},
	}
}

type Storage struct {
	Orders interface {
		Create(context.Context, *Order) error
		GetById(context.Context, string) (*Order, error)
		Cancel(context.Context, string) error
		Confirm(context.Context, string) error
		GetByIdempotencyKey(context.Context, string, string) (*Order, error)
	}

	Products interface {
		Create(context.Context, *Product) error
		GetById(context.Context, string) (*Product, error)
	}
}
