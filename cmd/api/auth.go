package main

import (
	"net/http"

	"github.com/myselfBZ/reserv-service/internal/auth"
	"github.com/myselfBZ/reserv-service/internal/store"
)

type UserWithToken struct {
	*store.User
	Tokens *auth.TokenPair `json:"token"`
}

type loginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type registerPayload struct {
	FirstName string `json:"first_name" validate:"required,min=2,max=50"`
	LastName  string `json:"last_name" validate:"min=2,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
}

func (a *api) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload registerPayload
	if err := readJSON(w, r, &payload); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		FirstName: payload.FirstName,
		Email:     payload.Email,
		Role: store.Role{
			Name: "user",
		},
	}

	if err := user.Password.Set(payload.Password); err != nil {
		a.internalServerError(w, r, err)
		return
	}

	pair, err := a.auth.GenerateTokenPair(
		user.Id.String(), 
		make(map[string]any),
	)

	if err != nil {
		a.internalServerError(w, r, err)
	}

	userWithToken := UserWithToken{
		User:  user,
		Tokens: pair,
	}
	if err := a.jsonResponse(w, http.StatusCreated, userWithToken); err != nil {
		a.internalServerError(w, r, err)
	}
}

func (a *api) loginHandler(w http.ResponseWriter, r *http.Request) {

}
