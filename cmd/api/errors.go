package main

import (
	"errors"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

var(
	ErrMalformedJsonPayload = errors.New("malformed json payload")
)

func (a *api) conflictResponse(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Errorw("conflict response", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusConflict, err.Error())
}

func (a *api) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Warnw("not found error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusNotFound, "not found")
}

func (a *api) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Errorw("internal error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusInternalServerError, "the server encountered a problem")
}

func (a *api) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Warnw("bad request", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusBadRequest, err.Error())
}

func (a *api) unauthorizedErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Warnw("unauthorized error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusUnauthorized, "unauthorized")
}

