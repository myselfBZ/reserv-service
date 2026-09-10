package store

import (
	"context"
	"time"
)

var	QueryTimeoutDuration = time.Second * 5 

type Storage struct {
	Users interface {
		Create(context.Context, *User) error
		GetByEmail(context.Context, string) (*User, error)
	}

	Orders interface {
		Place(context.Context)
		Get(context.Context)
		Update(context.Context)
		Delete(context.Context)
	}

	Products interface {
		Create(context.Context)
		Get(context.Context)
		Delete(context.Context)
	}
}
