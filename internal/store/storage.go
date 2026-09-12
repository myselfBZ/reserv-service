package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrResourceNotFound = errors.New("resource not found")
	QueryTimeoutDuration = time.Second * 5
)

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		Roles: &RoleStore{db: db},
		Users: &UserStore{db: db},
		Orders:   &OrderStore{db: db},
		Products: &ProductStore{db: db},
		Health: &Health{db: db},
	}
}

type Storage struct {
	Roles interface {
		GetByName(context.Context, string) (*Role, error)
	}

	Users interface {
		Create(context.Context, *User) error
		GetById(context.Context, string) (*User, error)
		GetByEmail(context.Context, string) (*User, error)
	}

	Orders interface {
		Create(context.Context, *Order) error
		GetById(context.Context, string) (*Order, error)
		Cancel(context.Context, string) error
		Confirm(context.Context, string) error
		GetByIdempotencyKey(context.Context, string, string) (*Order, error)
		CancelStale(ctx context.Context) ([]string, error) 
	}

	Products interface {
		Create(context.Context, *Product) error
		GetById(context.Context, string) (*Product, error)
	}


	Health interface {
		Get() (*HealthInfo, error)
	}
}
