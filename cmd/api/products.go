package main

import (
	"errors"
	"net/http"

	"github.com/myselfBZ/reserv-service/internal/store"
)

type createProductPayload struct {
	Name          string  `json:"name" validate:"required,max=255"`
	Price         float64 `json:"price" validate:"required,gt=0"`
	StockQuantity int     `json:"stock_quantity" validate:"required,gte=0"`
}

// CreateProduct godoc
//
//	@Summary		Creates a product
//	@Description	Creates a new product in the inventory. Admin-only.
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Param			product	body		createProductPayload	true	"Product details"
//	@Success		201		{object}	store.Product
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationError	
//	@Failure		500		{object}	ErrorResponse
//	@Security		ApiKeyAuth
//	@Router			/products/ [post]
func (a *api) createProductHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	isAdmin, err := a.checkRolePrecedence(r.Context(), user, "admin")
	if err != nil {
		a.internalServerError(w, r, err)
		return
	}

	if !isAdmin {
		a.unauthorizedErrorResponse(w, r, errors.New("unauthorized attempt to product creation"))
		return
	}

	var p createProductPayload
	if err := readJSON(w, r, &p); err != nil {
		a.logger.Warnw("malformed json payload", "err", err)
		writeJSONError(w, http.StatusBadRequest, "malformed json payload")
		return
	}

	if err := Validate.Struct(&p); err != nil {
		a.logger.Warnw("payload failed the validation", "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, &ValidationError{
			Message: "validation failed",
			Details: formatValidationErrors(err),
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
		return
	}

	writeJSON(w, http.StatusCreated, product)
}
