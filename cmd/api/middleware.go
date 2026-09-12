package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/myselfBZ/reserv-service/internal/store"
)

const userCtx = "user"

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

		claims, ok := jwtToken.Claims.(jwt.MapClaims)
		
		if !ok {
			a.unauthorizedErrorResponse(w, r, errors.New("invalid claims"))
			return
		}

		userId, ok := claims["sub"].(string)
		if !ok {
			a.unauthorizedErrorResponse(w, r, errors.New("invalid user id"))
			return
		}

		user, err := a.getUser(r.Context(), userId)

		if err != nil {
			switch err {
			case store.ErrResourceNotFound:
				a.logger.Warnw("user not found by id", "err", err)
				writeJSONError(w, http.StatusNotFound, "user not found")
			default:
				a.internalServerError(w, r, err)
			}
			return
		}

		ctx := context.WithValue(r.Context(), userCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *api) checkRolePrecedence(ctx context.Context, user *store.User, roleName string) (bool, error) {
	role, err := a.store.Roles.GetByName(ctx, roleName)
	if err != nil {
		return false, err
	}

	return user.Role.Level >= role.Level, nil
}
