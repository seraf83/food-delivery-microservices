// Package httpapi exposes the delivery-service REST endpoints. Paths and
// response shapes match the former Symfony DeliveryController.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/seraf83/food-delivery/delivery-service/internal/events"
	"github.com/seraf83/food-delivery/delivery-service/internal/store"
)

const timeLayout = "2006-01-02 15:04:05"

type Store interface {
	CompleteDelivery(ctx context.Context, orderID int64, at time.Time) (store.Delivery, error)
	DeliveryByOrder(ctx context.Context, orderID int64) (store.Delivery, error)
	Couriers(ctx context.Context) ([]store.Courier, error)
}

type Publisher interface {
	Publish(ctx context.Context, typ string, data any) error
}

type API struct {
	store Store
	pub   Publisher
	log   *slog.Logger
}

func New(s Store, p Publisher, log *slog.Logger) http.Handler {
	a := &API{store: s, pub: p, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /deliveries/{orderId}/complete", a.complete)
	mux.HandleFunc("GET /deliveries/{orderId}", a.status)
	mux.HandleFunc("GET /couriers", a.couriers)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

func (a *API) complete(w http.ResponseWriter, r *http.Request) {
	orderID, ok := orderIDParam(w, r)
	if !ok {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	d, err := a.store.CompleteDelivery(r.Context(), orderID, now)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	case errors.Is(err, store.ErrAlreadyDone):
		// Idempotent: don't free the courier or publish the event twice.
		writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
		return
	case err != nil:
		a.fail(w, "complete delivery", err)
		return
	}

	err = a.pub.Publish(r.Context(), events.TypeDeliveryCompleted, events.DeliveryCompleted{
		OrderID:     orderID,
		UserID:      d.UserID,
		CompletedAt: now.Format(timeLayout),
	})
	if err != nil {
		a.fail(w, "publish DeliveryCompleted", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

func (a *API) status(w http.ResponseWriter, r *http.Request) {
	orderID, ok := orderIDParam(w, r)
	if !ok {
		return
	}

	d, err := a.store.DeliveryByOrder(r.Context(), orderID)
	if err == nil && !canView(r, d) {
		err = store.ErrNotFound // don't reveal that someone else's order exists
	}
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	if err != nil {
		a.fail(w, "get delivery", err)
		return
	}

	var completedAt *string
	if d.CompletedAt != nil {
		s := d.CompletedAt.Format(timeLayout)
		completedAt = &s
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":           d.ID,
		"order_id":     d.OrderID,
		"courier_id":   d.CourierID,
		"status":       d.Status,
		"completed_at": completedAt,
		"created_at":   d.CreatedAt.Format(timeLayout),
	})
}

func (a *API) couriers(w http.ResponseWriter, r *http.Request) {
	couriers, err := a.store.Couriers(r.Context())
	if err != nil {
		a.fail(w, "list couriers", err)
		return
	}
	writeJSON(w, http.StatusOK, couriers)
}

// canView limits customers to their own deliveries. The gateway sets the
// X-User-* headers from the JWT; requests without them are internal calls.
func canView(r *http.Request, d store.Delivery) bool {
	if r.Header.Get("X-User-Role") != "ROLE_CUSTOMER" {
		return true
	}
	return r.Header.Get("X-User-Id") == strconv.FormatInt(d.UserID, 10)
}

func (a *API) fail(w http.ResponseWriter, op string, err error) {
	a.log.Error(op, "err", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal error"})
}

func orderIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("orderId"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
