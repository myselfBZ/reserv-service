package main

import "time"

type dbCfg struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type authCfg struct {
	accessSecret  string
	refreshSecret string
	iss           string
	aud           string
	exp           time.Duration
}

type config struct {
	env  string
	addr string
	db   dbCfg
	auth authCfg
}
