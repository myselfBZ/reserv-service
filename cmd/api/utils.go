package main

import (
	"net/http"

	"github.com/google/uuid"
)


func getUserId(r *http.Request) (uuid.UUID, error) {
	id := r.Context().Value("user-id").(string)
	return uuid.Parse(id)
}
