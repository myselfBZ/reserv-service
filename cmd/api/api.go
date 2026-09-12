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
	"github.com/myselfBZ/reserv-service/internal/auth"
	"github.com/myselfBZ/reserv-service/internal/env"
	"github.com/myselfBZ/reserv-service/internal/store"
	"github.com/myselfBZ/reserv-service/internal/store/cache"
	"go.uber.org/zap"
)

type api struct {
	cfg       config
	startedAt time.Time
	cache     *cache.Cache
	auth      auth.Authenticator
	store     *store.Storage
	logger    *zap.SugaredLogger

	stop chan struct{}
}

func (a *api) mount() http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
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
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", a.registerUserHandler)
			r.Post("/login", a.loginHandler)
		})

		r.Route("/products", func(r chi.Router) {
			r.Use(a.AuthTokenMiddleware)
			r.Post("/", a.createProductHandler)
		})

		r.Route("/orders", func(r chi.Router) {
			r.Use(a.AuthTokenMiddleware)
			r.Post("/", a.placeOrderHandler)

			r.Route("/{id}", func(r chi.Router) {
				r.Use(a.ordersContextMiddleware)
				r.Get("/", a.checkOrderOwnership("admin", a.getOrderByIdHandler))
				r.Post("/cancel", a.checkOrderOwnership("admin", a.cancelOrderHandler))
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(a.AuthTokenMiddleware)
			r.Get("/health", a.healthCheckHandler)
		})
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
