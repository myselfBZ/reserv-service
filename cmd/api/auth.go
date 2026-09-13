package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/myselfBZ/reserv-service/internal/store"
)

type UserWithToken struct {
	*store.User
	Token string `json:"token"`
}

type loginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type registerPayload struct {
	FirstName string `json:"first_name" validate:"required,min=2,max=50"`
	LastName  string `json:"last_name" validate:"max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
}

// RegisterUser godoc
//
//	@Summary		Registers a new user
//	@Description	Creates a new user account with the default "user" role, issues an access token in the response body, and sets a refresh token as an HttpOnly cookie scoped to /v1/auth/refresh
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			user	body		registerPayload	true	"Registration details"
//	@Success		201		{object}	UserWithToken
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/auth/register [post]
func (a *api) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var p registerPayload
	if err := readJSON(w, r, &p); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(p); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		FirstName: p.FirstName,
		Email:     p.Email,
		Role: store.Role{
			Name: "user",
		},
	}

	if err := user.Password.Set(p.Password); err != nil {
		a.internalServerError(w, r, err)
		return
	}

	if err := a.store.Users.Create(r.Context(), user); err != nil {
		switch err {
		case store.ErrDuplicateEmail:
			a.conflictResponse(w, r, err)
		default:
			a.internalServerError(w, r, err)
		}
		return
	}

	pair, err := a.auth.GenerateTokenPair(
		user.Id.String(),
		make(map[string]any),
	)

	if err != nil {
		a.internalServerError(w, r, err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh-token",
		Value:    pair.RefreshToken,
		Path:     "/v1/auth/refresh",
		HttpOnly: true,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
	})

	userWithToken := UserWithToken{
		User:  user,
		Token: pair.AccessToken,
	}

	writeJSON(w, http.StatusCreated, userWithToken)
}

// LoginUser godoc
//
//	@Summary		Logs in a user
//	@Description	Authenticates a user by email and password, issues an access token in the response body, and sets a refresh token as an HttpOnly cookie scoped to /v1/auth/refresh
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		loginPayload	true	"Login credentials"
//	@Success		200			{object}	UserWithToken
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/auth/login [post]
func (a *api) loginHandler(w http.ResponseWriter, r *http.Request) {
	var p loginPayload
	if err := readJSON(w, r, &p); err != nil {
		a.badRequestResponse(w, r, ErrMalformedJsonPayload)
		return
	}

	if err := Validate.Struct(p); err != nil {
		a.logger.Warnw("loginPayload failed on validation", "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, &ValidationError{
			Message: "validation failed",
			Details: formatValidationErrors(err),
		})
		return
	}

	user, err := a.store.Users.GetByEmail(r.Context(), p.Email)
	if err != nil {
		switch err {
		case store.ErrResourceNotFound:
			a.notFoundResponse(w, r, err)
		default:
			a.internalServerError(w, r, err)
		}
		return
	}

	if err := user.Password.Compare(p.Password); err != nil {
		a.unauthorizedErrorResponse(w, r, err)
		return
	}

	pair, err := a.auth.GenerateTokenPair(user.Id.String(), make(map[string]any))
	if err != nil {
		a.internalServerError(w, r, err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh-token",
		Value:    pair.RefreshToken,
		Path:     "/v1/auth/refresh",
		HttpOnly: true,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
	})

	userWithToken := UserWithToken{
		User:  user,
		Token: pair.AccessToken,
	}

	writeJSON(w, http.StatusOK, userWithToken)
}

// RefreshToken godoc
//
//	@Summary		Refreshes an access token
//	@Description	Validates the refresh token cookie and issues a new access/refresh token pair, rotating the refresh token cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		201	{object}	UserWithToken
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Route 			/auth/refresh 	[post]
func (a *api) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("refresh-token")
	if err != nil {
		a.unauthorizedErrorResponse(w, r, err)
		return
	}
	tok, err := a.auth.ValidateRefreshToken(c.Value)
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:   "refresh-token",
			Value:  "",
			MaxAge: -1,
		})
		a.unauthorizedErrorResponse(w, r, fmt.Errorf("invalid token"))
		return
	}
	userId, err := a.auth.ExtractUserID(tok)
	if err != nil {
		a.unauthorizedErrorResponse(w, r, fmt.Errorf("invalid token claims"))
		return
	}
	user, err := a.getUser(r.Context(), userId)
	if err != nil {
		switch err {
		case store.ErrResourceNotFound:
			a.logger.Warnw("user not found", "err", err)
			writeJSONError(w, http.StatusNotFound, "user not found")
		default:
			a.internalServerError(w, r, err)
		}
		return
	}

	pair, err := a.auth.GenerateTokenPair(user.Id.String(), make(map[string]any))
	if err != nil {
		a.internalServerError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh-token",
		Value:    pair.RefreshToken,
		Path:     "/v1/auth/refresh",
		HttpOnly: true,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
	})

	userWithToken := UserWithToken{
		User:  user,
		Token: pair.AccessToken,
	}

	writeJSON(w, http.StatusCreated, userWithToken)
}
