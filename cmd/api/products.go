package main

import (
	"net/http"

	"github.com/myselfBZ/reserv-service/internal/store"
)

type createProductPayload struct {
	Name          string  `json:"name" validate:"required,max=255"`
	Price         float64 `json:"price" validate:"required,gt=0"`
	StockQuantity int     `json:"stock_quantity" validate:"required,gte=0"`
}

func (a *api) createProductHandler(w http.ResponseWriter, r *http.Request) {
	var p createProductPayload
	if err := readJSON(w, r, &p); err != nil {
		a.logger.Warnw("malformed json payload", "err", err)
		writeJSONError(w, http.StatusBadRequest, "malformed json payload")
		return
	}

	if err := Validate.Struct(&p); err != nil {
		a.logger.Warnw("payload failed the validation", "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":"json validation failed",
			"details": formatValidationErrors(err),
		})
		return
	}

	product := &store.Product{
		Name:          p.Name,
		Price:         p.Price,
		StockQuantity: p.StockQuantity,
	}

	if err := a.store.Products.Create(r.Context(), product); err != nil {
		a.logger.Errorw("could not create a product", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
	}

	writeJSON(w, http.StatusCreated, product)
}
