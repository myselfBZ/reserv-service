package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/myselfBZ/reserv-service/internal/store"
	"github.com/myselfBZ/reserv-service/internal/store/cache"
)


var(
	janitorInterval = time.Minute * 5
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
	user := getUserFromContext(r)
	items := []store.OrderItem{}
	for _, it := range p.Items {
		items = append(items, store.OrderItem{
			ProductId: it.ProductId,
			Quantity: it.Quantity,
		})
	}
	ordr := &store.Order{
		UserId:         user.Id.String(),
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
			writeJSONError(w, http.StatusConflict, "insuffcient stock amount")
		case store.ErrDuplicateIdempotencyKey:
			ordr, err := a.store.Orders.GetByIdempotencyKey(r.Context(), user.Id.String(), idempKey)
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

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
		defer cancel()
		if err := a.cache.Orders.Delete(ctx, orderId); err != nil {
			a.logger.Errorw("order cahce del failed", "err", err)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"message":"success",
		"order_id":orderId,
	})
}

func (a *api) getOrderByIdHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	isAdmin, err := a.checkRolePrecedence(r.Context(), user, "admin")
	if err != nil {
		a.internalServerError(w, r, err)
	}
	id := r.PathValue("id")
	validId, err := uuid.Parse(id)
	if err != nil {
		a.logger.Warnw("invalid order uuid", "err", err)
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid order id")
		return
	}

	o, err := a.getOrder(r.Context(), validId.String()) 

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

	if o.UserId != user.Id.String() && !isAdmin {
		a.unauthorizedErrorResponse(w, r, fmt.Errorf("order ownership check failed"))
		return
	}

	writeJSON(w, http.StatusOK, o)
}

func (a *api) cancelStaleOrdersJanitor() {
	t := time.NewTicker(janitorInterval)

	for {
		select {
		case <- t.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
			ids, err := a.store.Orders.CancelStale(ctx)
			if err != nil {
				a.logger.Errorw("cancelStaleOrders failed to cancel orders", "err", err)
				cancel()
				continue
			}
			a.cache.Orders.Delete(ctx, ids...)
			cancel()
		case <- a.stop:
			t.Stop()
			a.logger.Infow("stale orders janitor has stopped")
			return
		}
	}
}

func (a *api) getOrder(ctx context.Context, id string) (*store.Order, error) {
	o, err := a.cache.Orders.GetById(ctx, id)

	if err == nil {
		return o, nil
	} else if err != cache.ErrNotFound {
		a.logger.Errorw("order cache error", "err", err)
	}

	o, err = a.store.Orders.GetById(ctx, id) 
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
		defer cancel()
		if err := a.cache.Orders.Set(ctx, o); err != nil {
			a.logger.Errorw("order cahce set failed", "err", err)
		}
	}()
	return o, err
}
