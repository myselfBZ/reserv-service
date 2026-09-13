package main

import (
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ValidationError struct {
	Message string       `json:"message"`
	Details []fieldError `json:"details"`
}

type fieldError struct {
	Field string `json:"field"`
	Msg   string `json:"message"`
}

func formatValidationErrors(err error) []fieldError {
	var errors []fieldError

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors
	}

	for _, err := range validationErrs {
		var message string

		switch err.Tag() {
		case "required":
			message = fmt.Sprintf("%s is required", err.Field())
		case "max":
			message = fmt.Sprintf("%s must not exceed %s characters", err.Field(), err.Param())
		case "gt", "gte":
			message = fmt.Sprintf("%s must be greater than %s", err.Field(), err.Param())
		case "min":
			message = fmt.Sprintf("%s must be at least %s", err.Field(), err.Param())
		case "uuid":
			message = fmt.Sprintf("%s must be a valid UUID", err.Field())
		default:
			message = fmt.Sprintf("%s failed validation on '%s'", err.Field(), err.Tag())
		}

		errors = append(errors, fieldError{
			Field: err.Field(),
			Msg:   message,
		})
	}

	return errors
}

func getUserId(r *http.Request) (uuid.UUID, error) {
	id := r.Context().Value("user-id").(string)
	return uuid.Parse(id)
}
