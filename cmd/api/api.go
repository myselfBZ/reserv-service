package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/myselfBZ/reserv-service/internal/env"
	"github.com/myselfBZ/reserv-service/internal/store"
	"github.com/myselfBZ/reserv-service/internal/store/cache"
	"go.uber.org/zap"
)

type api struct {
	cfg    config

	cache  *cache.Cache
	store  *store.Storage
	logger *zap.SugaredLogger

	stop   chan struct{}
}

func (a *api) mount() http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		// Sus...
		AllowedOrigins:   []string{env.MustGetString("CORS_ALLOWED_ORIGIN")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Post("/products", a.createProductHandler)

		r.Post("/orders", a.placeOrderHandler)
		r.Get("/orders/{id}", a.getOrderByIdHandler)
		r.Post("/orders/{id}/cancel", a.cancelOrderHandler)
	})

	return r
}

func (a *api) run(mux http.Handler) error {
	srv := &http.Server{
		Addr:         a.cfg.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	shutdown := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		a.logger.Infow("signal caught", "signal", s.String())

		shutdown <- srv.Shutdown(ctx)
	}()

	// Background jobs
	go a.cancelStaleOrdersJanitor()

	a.logger.Infow("server has started", "addr", a.cfg.addr, "env", a.cfg.env)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdown
	if err != nil {
		return err
	}
	
	a.stop <- struct{}{}

	a.logger.Infow("server has stopped", "addr", a.cfg.addr, "env", a.cfg.env)

	return nil
}
