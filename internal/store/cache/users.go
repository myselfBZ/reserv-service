package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/myselfBZ/reserv-service/internal/store"
)


const (
	userPrefix = "user:%s"
	UserExpTime = time.Minute * 5
) 


type UserStore struct {
	client *redis.Client 
}

func (s *UserStore) Get(ctx context.Context, id string) (*store.User, error) {
	key := fmt.Sprintf(userPrefix, id)
	data, err := s.client.Get(ctx, key).Result()
	if err  != nil {
		if err == redis.Nil {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var u store.User
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserStore) Set(ctx context.Context, u *store.User) error {
	key := fmt.Sprintf(userPrefix, u.Id)
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, UserExpTime).Err()
}

func (s *UserStore) Del(ctx context.Context, id string) error {
	key := fmt.Sprintf(userPrefix, id)
	return s.client.Del(ctx, key).Err()
}
