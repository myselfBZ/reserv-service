package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)
var(
	ErrResourceNotFound = errors.New("resource not found")
)
var	QueryTimeoutDuration = time.Second * 5 

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		Orders: &OrderStore{db: db},
		Users: &UserStore{db: db},
		Products: &ProductStore{db: db},
	}
}

type Storage struct {
	Users interface {
		Create(context.Context, *User) error
		GetByEmail(context.Context, string) (*User, error)
	}

	Orders interface {
		Create(context.Context, *Order) error
		GetById(context.Context, string) (*Order, error)
		Delete(context.Context, string) error
	}

	Products interface {
		Create(context.Context, *Product) error
		GetById(context.Context, string) (*Product, error)
	}
}
