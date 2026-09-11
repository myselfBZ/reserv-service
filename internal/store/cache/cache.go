package cache

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/myselfBZ/reserv-service/internal/store"
)


func New(c *redis.Client) *Cache {
	return &Cache{
		Orders: &OrdersStore{client: c},
	}
}

type Cache struct {
	Orders interface {
		Set(ctx context.Context, o *store.Order) error
		GetById(ctx context.Context, id string) (*store.Order, error)
		Delete(ctx context.Context, id ...string) error
	}

	// Products interface {
	// 	Set(ctx context.Context, p *store.Product) error
	// 	GetById(ctx context.Context, id string) (*store.Product, error)
	// 	Delete(ctx context.Context, id string) error
	// }
}
