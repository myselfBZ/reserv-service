package main

type dbCfg struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type authCfg struct {
	secret string
	iss    string
	aud    string
}

type config struct {
	env  string
	addr string
	db   dbCfg
	auth authCfg
}
