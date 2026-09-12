package main

import (
	"context"
	"net/http"
	"time"

	"github.com/myselfBZ/reserv-service/internal/store"
	"github.com/myselfBZ/reserv-service/internal/store/cache"
)

func (a *api) getUser(ctx context.Context, userId string) (*store.User, error) {
	u, err := a.cache.Users.Get(ctx, userId)
	if err == nil {
		a.logger.Info("cache hit for user")
		return u, nil
	} else if err != cache.ErrNotFound {
		a.logger.Errorf("user retrieval from cache failed", "err", err)
	}

	u, err = a.store.Users.GetById(ctx, userId)
	if err != nil {
		return nil, err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
		defer cancel()
		if err := a.cache.Users.Set(ctx, u); err != nil {
			a.logger.Errorf("caching user failed", "err", err)
		}
	}()
	return u, nil
}

func getUserFromContext(r *http.Request) *store.User {
	user, _ := r.Context().Value(userCtx).(*store.User)
	return user
}
