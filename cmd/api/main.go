package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/myselfBZ/reserv-service/internal/db"
	"github.com/myselfBZ/reserv-service/internal/env"
	"github.com/myselfBZ/reserv-service/internal/store"
	"github.com/myselfBZ/reserv-service/internal/store/cache"
	"go.uber.org/zap"
)

func main() {
	a := &api{
		cfg: config{
			env:  env.GetString("ENV", "development"),
			addr: env.GetString("ADDR", ":8080"),
			db: dbCfg{
				addr:         env.MustGetString("DB"),
				maxOpenConns: env.GetInt("MAX_OPEN_CONNS", 30),
				maxIdleConns: env.GetInt("MAX_IDLE_CONNS", 30),
				maxIdleTime:  env.GetString("MAX_IDLE_TIME", "15m"),
			},
			auth: authCfg{
				secret: env.MustGetString("AUTH_SECRET"),
				iss:    env.MustGetString("AUTH_ISS"),
				aud:    env.MustGetString("AUTH_AUD"),
			},
			cacheCfg: redisConfig{
				addr:    env.GetString("REDIS_ADDR", "localhost:6379"),
				pw:      env.GetString("REDIS_PW", ""),
				db:      env.GetInt("REDIS_DB", 0),
			},
		},
	}

	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	a.logger = logger

	db, err := db.New(
		a.cfg.db.addr, 
		a.cfg.db.maxOpenConns,
		a.cfg.db.maxIdleConns,
		a.cfg.db.maxIdleTime,
	)
	if err != nil {
		logger.Fatalw("database connection failed", "err", err)
	}
	s := store.NewStorage(db)
	a.store = s

	rdClient := cache.NewRedisClient(
		a.cfg.cacheCfg.addr, 
		a.cfg.cacheCfg.pw, 
		a.cfg.cacheCfg.db,
	)
	a.cache = cache.New(rdClient)

	mux := a.mount()
	logger.Fatal(a.run(mux))
}
