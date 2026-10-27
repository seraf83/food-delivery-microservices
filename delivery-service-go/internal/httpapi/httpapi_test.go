package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seraf83/food-delivery/delivery-service/internal/store"
)

type fakeStore struct{ d store.Delivery }

func (f fakeStore) CompleteDelivery(context.Context, int64, time.Time) (store.Delivery, error) {
	return f.d, nil
}

func (f fakeStore) DeliveryByOrder(_ context.Context, orderID int64) (store.Delivery, error) {
	if orderID != f.d.OrderID {
		return store.Delivery{}, store.ErrNotFound
	}
	return f.d, nil
}

func (f fakeStore) Couriers(context.Context) ([]store.Courier, error) { return nil, nil }

func TestStatusVisibility(t *testing.T) {
	h := New(fakeStore{d: store.Delivery{ID: 1, OrderID: 25, UserID: 5, Status: "assigned"}}, nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)))

	cases := []struct {
		name, userID, role string
		want               int
	}{
		{"owner", "5", "ROLE_CUSTOMER", http.StatusOK},
		{"other customer", "6", "ROLE_CUSTOMER", http.StatusNotFound},
		{"courier", "6", "ROLE_COURIER", http.StatusOK},
		{"internal call", "", "", http.StatusOK},
	}
	for _, c := range cases {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/deliveries/25", nil)
		if c.role != "" {
			req.Header.Set("X-User-Id", c.userID)
			req.Header.Set("X-User-Role", c.role)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, rec.Code, c.want)
		}
	}
}
