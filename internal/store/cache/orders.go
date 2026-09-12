package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/myselfBZ/reserv-service/internal/store"
)



const OrderExpTime = time.Minute * 5
var (
	orderPrefix = "order:%s"
)

type OrdersStore struct {
	client *redis.Client
}

func (s *OrdersStore) GetById(ctx context.Context, id string) (*store.Order, error) {
	key := fmt.Sprintf(orderPrefix, id)
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var o store.Order
	if err := json.Unmarshal([]byte(data), &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *OrdersStore) Delete(ctx context.Context, ids ...string) error {
	keys := []string{}
	for _, id := range ids {
		keys = append(keys, fmt.Sprintf(orderPrefix, id))
	}
	_, err := s.client.Del(ctx, keys...).Result()
	return err
}

func (s *OrdersStore) Set(ctx context.Context, o *store.Order) error {
	key := fmt.Sprintf(orderPrefix, o.Id)
	byteData, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, byteData, OrderExpTime).Err()
}
