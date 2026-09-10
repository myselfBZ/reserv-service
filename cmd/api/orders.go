package main

import (
	"net/http"

	"github.com/myselfBZ/reserv-service/internal/store"
)

type placeOrderPayload struct {
	Items []itemPayload `json:"items"`
}

type itemPayload struct {
	ProductId string `json:"product_id" validate:"required"`
	Quantity  int `json:"quantity" validate:"required"`
}

func (a *api) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	idempKey := r.Header.Get("Idempotency-Key")
	if idempKey == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "Idempotency-Key is missing in the headers")
		a.logger.Warnw("Idempotency-Key missing")
		return
	}
	var p placeOrderPayload
	if err := readJSON(w, r, &p); err != nil {
		writeJSONError(w, http.StatusBadRequest, "malformed json payload")
		a.logger.Warnw("malformed json payload", "err", err)
		return
	}
	userId := "33745878-a505-4785-8318-f2c86b61d1bc"
	// userId, err := getUserId(r)
	// if err != nil {
	// 	writeJSONError(w, http.StatusUnauthorized, "invalid user id")
	// 	a.logger.Warnw("invalid user id", "err", err)
	// 	return
	// }
	items := []store.OrderItem{}
	for _, it := range p.Items {
		items = append(items, store.OrderItem{
			ProductId: it.ProductId,
			Quantity: it.Quantity,
		})
	}
	ordr := &store.Order{
		UserId:         userId,
		IdempotencyKey: idempKey,
		OrderItems: items,
	}

	if err := a.store.Orders.Create(r.Context(), ordr); err != nil {
		switch err {
		case store.ErrInvalidProductId:
			a.logger.Warnw("order to a non-existing product", "err", err)
			writeJSONError(w, http.StatusNotFound, "product is not found")
		case store.ErrInsufficientStock:
			a.logger.Warnw("insuffcient stock amount", "err", err)
			writeJSONError(w, http.StatusBadRequest, "insuffcient stock amount")
		case store.ErrDuplicateIdempotencyKey:
			ordr, err := a.store.Orders.GetByIdempotencyKey(r.Context(), userId, idempKey)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "server encountered an error")
				a.logger.Errorw("order not found by idempotency key", "err", err)
			}
			writeJSON(w, http.StatusCreated, ordr)
			return
		default:
			a.logger.Errorw("internal server error", "err", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, ordr)
}
