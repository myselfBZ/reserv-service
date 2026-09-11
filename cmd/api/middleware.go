package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func (a *api) AuthTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			a.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is missing"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			a.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is malformed"))
			return
		}

		token := parts[1]
		jwtToken, err := a.auth.ValidateAccessToken(token)
		if err != nil {
			a.unauthorizedErrorResponse(w, r, err)
			return
		}

		claims, _ := jwtToken.Claims.(jwt.MapClaims)

		userId, ok := claims["sub"]
		if !ok {
			a.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx := r.Context()

		ctx = context.WithValue(ctx, "user-id", userId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

