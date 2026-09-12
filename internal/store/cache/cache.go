package cache

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/myselfBZ/reserv-service/internal/store"
)

var ErrNotFound = errors.New("resource is not present in cache") 

func New(c *redis.Client) *Cache {
	return &Cache{
		Orders: &OrdersStore{client: c},
		Users: &UserStore{client: c},
	}
}

type Cache struct {
	Orders interface {
		Set(ctx context.Context, o *store.Order) error
		GetById(ctx context.Context, id string) (*store.Order, error)
		Delete(ctx context.Context, id ...string) error
	}

	Users interface {
		Set(ctx context.Context, u *store.User) error
		Get(ctx context.Context, id string) (*store.User, error)
		Del(ctx context.Context, id string) error
	}
}
