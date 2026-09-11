package main

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/myselfBZ/reserv-service/internal/store"
)

type placeOrderPayload struct {
	Items []itemPayload `json:"items" validate:"required,min=1,dive"`
}

type itemPayload struct {
	ProductId string `json:"product_id" validate:"required,uuid"`
	Quantity  int `json:"quantity" validate:"required,gt=0"`
}

func (a *api) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	idempKey := r.Header.Get("Idempotency-Key")
	if idempKey == "" {
		a.logger.Warnw("Idempotency-Key missing")
		writeJSONError(w, http.StatusUnprocessableEntity, "Idempotency-Key is missing in the headers")
		return
	}
	var p placeOrderPayload
	if err := readJSON(w, r, &p); err != nil {
		writeJSONError(w, http.StatusBadRequest, "malformed json payload")
		a.logger.Warnw("malformed json payload", "err", err)
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


func (a *api) cancelOrderHandler(w http.ResponseWriter, r *http.Request) {
	orderId := r.PathValue("id")
	validId, err := uuid.Parse(orderId)
	if err != nil {
		a.logger.Warnw("invalid uuid", "id", orderId)
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid order id")
		return
	}

	if err := a.store.Orders.Cancel(r.Context(), validId.String()); err != nil {
		switch err {
		case store.ErrResourceNotFound:
			a.logger.Warnw("order not found for cancellation", "err", err)
			writeJSONError(w, http.StatusNotFound, "order not found")
		case store.ErrOrderCannotBeCancelled:
			writeJSONError(w, http.StatusBadRequest, err.Error())
			a.logger.Warnw("order cannot be cancelled", "err", err)
		default:
			a.logger.Errorw("internal server error order cancellation failed", "err", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":"success",
		"order_id":orderId,
	})
}


func (a *api) getOrderById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	validId, err := uuid.Parse(id)
	if err != nil {
		a.logger.Warnw("invalid order uuid", "err", err)
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid order id")
		return
	}
	 o, err := a.store.Orders.GetById(r.Context(), validId.String()) 

	 if err != nil {
		 switch err {
		 case store.ErrResourceNotFound:
			 a.logger.Warnw("order not found for fetching", "err", err)
			 writeJSONError(w, http.StatusNotFound, "order not found")
		 default:
			 a.logger.Errorw("internal server error order not fetched", "err", err)
			 writeJSONError(w, http.StatusInternalServerError, "internal server error")
		 }
		 return
	 }

	 writeJSON(w, http.StatusOK, o)
}

