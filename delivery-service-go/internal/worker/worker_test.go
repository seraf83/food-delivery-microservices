package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/seraf83/food-delivery/delivery-service/internal/events"
	"github.com/seraf83/food-delivery/delivery-service/internal/store"
)

const orderPlaced = `{"type":"App\\Message\\OrderPlaced","data":{"orderId":5,"userId":9,"deliveryAddress":"Khreshchatyk 1","items":[],"totalAmount":250.5}}`

type assignCall struct {
	orderID, userID int64
	address         string
}

type fakeStore struct {
	courier store.Courier
	created bool
	err     error
	calls   []assignCall
}

func (f *fakeStore) AssignCourier(_ context.Context, orderID, userID int64, address string) (store.Courier, bool, error) {
	f.calls = append(f.calls, assignCall{orderID, userID, address})
	return f.courier, f.created, f.err
}

type published struct {
	typ  string
	data any
}

type fakePublisher struct {
	err  error
	sent []published
}

func (f *fakePublisher) Publish(_ context.Context, typ string, data any) error {
	f.sent = append(f.sent, published{typ, data})
	return f.err
}

// fakeReader serves msgs in order. Once they run out it cancels the worker's
// context, the same way SIGTERM would.
type fakeReader struct {
	msgs      []kafka.Message
	committed []int64
	closed    bool
	stop      context.CancelFunc
}

func (f *fakeReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if len(f.msgs) == 0 {
		f.stop()
		return kafka.Message{}, ctx.Err()
	}
	m := f.msgs[0]
	f.msgs = f.msgs[1:]
	return m, nil
}

func (f *fakeReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	for _, m := range msgs {
		f.committed = append(f.committed, m.Offset)
	}
	return nil
}

func (f *fakeReader) Close() error {
	f.closed = true
	return nil
}

func newWorker(st *fakeStore, pub *fakePublisher) *Worker {
	return &Worker{store: st, pub: pub, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestHandleAssignsCourierAndPublishes(t *testing.T) {
	st := &fakeStore{courier: store.Courier{ID: 3, Name: "Ivan", Phone: "+380501112233"}, created: true}
	pub := &fakePublisher{}

	if err := newWorker(st, pub).handle(t.Context(), kafka.Message{Value: []byte(orderPlaced)}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if want := []assignCall{{5, 9, "Khreshchatyk 1"}}; len(st.calls) != 1 || st.calls[0] != want[0] {
		t.Fatalf("AssignCourier calls = %+v, want %+v", st.calls, want)
	}
	if len(pub.sent) != 1 {
		t.Fatalf("published %d messages, want 1", len(pub.sent))
	}
	if pub.sent[0].typ != events.TypeDeliveryAssigned {
		t.Errorf("type = %q, want %q", pub.sent[0].typ, events.TypeDeliveryAssigned)
	}

	got, ok := pub.sent[0].data.(events.DeliveryAssigned)
	if !ok {
		t.Fatalf("data is %T, want events.DeliveryAssigned", pub.sent[0].data)
	}
	if got.EstimatedMinutes < 1 || got.EstimatedMinutes > 3 {
		t.Errorf("EstimatedMinutes = %d, want 1..3", got.EstimatedMinutes)
	}
	got.EstimatedMinutes = 0
	want := events.DeliveryAssigned{OrderID: 5, CourierID: 3, CourierName: "Ivan", CourierPhone: "+380501112233", UserID: 9}
	if got != want {
		t.Errorf("DeliveryAssigned = %+v, want %+v", got, want)
	}
}

func TestHandleSkipsWithoutPublishing(t *testing.T) {
	cases := []struct {
		name      string
		msg       string
		st        fakeStore
		wantStore bool // whether AssignCourier should be reached
	}{
		{"not JSON", `not json`, fakeStore{}, false},
		{"empty type", `{"data":{}}`, fakeStore{}, false},
		{"other event", `{"type":"App\\Message\\DeliveryCompleted","data":{}}`, fakeStore{}, false},
		{"bad payload", `{"type":"App\\Message\\OrderPlaced","data":{"orderId":"x"}}`, fakeStore{}, false},
		{"duplicate order", orderPlaced, fakeStore{created: false}, true},
		{"no couriers", orderPlaced, fakeStore{err: store.ErrNoCouriers}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pub := &fakePublisher{}
			if err := newWorker(&c.st, pub).handle(t.Context(), kafka.Message{Value: []byte(c.msg)}); err != nil {
				t.Fatalf("handle: %v, want nil (message should be skipped and committed)", err)
			}
			if reached := len(c.st.calls) > 0; reached != c.wantStore {
				t.Errorf("AssignCourier called = %v, want %v", reached, c.wantStore)
			}
			if len(pub.sent) != 0 {
				t.Errorf("published %d messages, want 0", len(pub.sent))
			}
		})
	}
}

func TestHandleReturnsTransientErrors(t *testing.T) {
	errDB := errors.New("connection refused")
	errKafka := errors.New("broker unavailable")

	cases := []struct {
		name string
		st   fakeStore
		pub  fakePublisher
		want error
	}{
		{"store fails", fakeStore{err: errDB}, fakePublisher{}, errDB},
		{"publish fails", fakeStore{created: true}, fakePublisher{err: errKafka}, errKafka},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := newWorker(&c.st, &c.pub).handle(t.Context(), kafka.Message{Value: []byte(orderPlaced)})
			if !errors.Is(err, c.want) {
				t.Errorf("handle error = %v, want %v", err, c.want)
			}
		})
	}
}

func TestRunCommitsOnlyHandledMessages(t *testing.T) {
	t.Run("commits each handled message and stops on cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := &fakeReader{stop: cancel, msgs: []kafka.Message{
			{Offset: 10, Value: []byte(orderPlaced)},
			{Offset: 11, Value: []byte(`garbage`)}, // skipped, but still committed
		}}
		w := newWorker(&fakeStore{created: true}, &fakePublisher{})
		w.reader = r

		if err := w.Run(ctx); err != nil {
			t.Fatalf("Run: %v, want nil on shutdown", err)
		}
		if len(r.committed) != 2 || r.committed[0] != 10 || r.committed[1] != 11 {
			t.Errorf("committed offsets = %v, want [10 11]", r.committed)
		}
		if !r.closed {
			t.Error("reader was not closed")
		}
	})

	t.Run("does not commit when handling fails", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		errDB := errors.New("connection refused")
		r := &fakeReader{stop: cancel, msgs: []kafka.Message{
			{Offset: 10, Value: []byte(orderPlaced)},
			{Offset: 11, Value: []byte(orderPlaced)},
		}}
		w := newWorker(&fakeStore{err: errDB}, &fakePublisher{})
		w.reader = r

		if err := w.Run(ctx); !errors.Is(err, errDB) {
			t.Fatalf("Run error = %v, want %v", err, errDB)
		}
		if len(r.committed) != 0 {
			t.Errorf("committed offsets = %v, want none (message must be redelivered)", r.committed)
		}
		if len(r.msgs) != 1 {
			t.Errorf("worker kept consuming after a failure: %d messages left, want 1", len(r.msgs))
		}
	})
}
