package main

import (
	"context"
	"log"
	"time"

	"github.com/myselfBZ/reserv-service/internal/db"
	"github.com/myselfBZ/reserv-service/internal/env"
	"github.com/myselfBZ/reserv-service/internal/store"
)

var (
	AdminEmail    string
	AdminPassword string
	DatabaseUrl   string
)

func init() {
	AdminEmail = env.MustGetString("ADMIN_EMAIL")
	AdminPassword = env.MustGetString("ADMIN_PASSWORD")
	DatabaseUrl = env.MustGetString("DB")
}

func main() {
	db, err := db.New(DatabaseUrl, 15, 15, "15m")
	if err != nil {
		log.Fatalf("db fail: %v", err)
	}
	s := store.NewStorage(db)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
	defer cancel()
	u := &store.User{
		FirstName: "Admin",
		Email: AdminEmail,
		Role: store.Role{
			Name: "admin",
		},
	}
	u.Password.Set(AdminPassword)
	if err := s.Users.Create(ctx, u); err != nil {
		log.Fatalf("failed to create an admin: %v", err)
	}
	log.Println("Successfully created an admin in the database")
}
