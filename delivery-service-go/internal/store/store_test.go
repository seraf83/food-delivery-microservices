package store

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

// testStore is shared by all tests in the package: starting MySQL takes
// several seconds, so one container is reused and tables are truncated per test.
var testStore *Store

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	flag.Parse()
	// Integration tests need Docker; `go test -short` skips them.
	if testing.Short() {
		return m.Run()
	}

	ctx := context.Background()
	ctr, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("delivery"),
		mysql.WithUsername("delivery_user"),
		mysql.WithPassword("delivery_pass"),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			fmt.Fprintln(os.Stderr, "terminate mysql container:", err)
		}
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, "start mysql container:", err)
		return 1
	}

	dsn, err := ctr.ConnectionString(ctx, "parseTime=true", "loc=UTC", "charset=utf8mb4")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mysql dsn:", err)
		return 1
	}
	if testStore, err = Open(ctx, dsn); err != nil {
		fmt.Fprintln(os.Stderr, "open store:", err)
		return 1
	}
	defer testStore.Close()

	if err := testStore.Migrate(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return m.Run()
}

// setup empties the tables and inserts couriers with the given names, all
// available. It returns their IDs in the same order.
func setup(t *testing.T, couriers ...string) []int64 {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs Docker")
	}

	ctx := t.Context()
	for _, q := range []string{"TRUNCATE TABLE deliveries", "TRUNCATE TABLE couriers"} {
		if _, err := testStore.db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	ids := make([]int64, 0, len(couriers))
	for i, name := range couriers {
		res, err := testStore.db.ExecContext(ctx,
			`INSERT INTO couriers (name, phone, status) VALUES (?, ?, 'available')`,
			name, fmt.Sprintf("+38050000%04d", i))
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		ids = append(ids, id)
	}
	return ids
}

func courierStatus(t *testing.T, id int64) string {
	t.Helper()
	var status string
	if err := testStore.db.QueryRowContext(t.Context(),
		`SELECT status FROM couriers WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestMigrateIsIdempotent(t *testing.T) {
	setup(t)
	if err := testStore.Migrate(t.Context()); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestAssignCourier(t *testing.T) {
	ids := setup(t, "Ivan", "Olena")
	ctx := t.Context()

	c, created, err := testStore.AssignCourier(ctx, 100, 7, "Khreshchatyk 1")
	if err != nil {
		t.Fatalf("AssignCourier: %v", err)
	}
	if !created {
		t.Error("created = false, want true for a new order")
	}
	if c.ID != ids[0] || c.Name != "Ivan" || c.Status != "busy" {
		t.Errorf("courier = %+v, want Ivan (id %d), busy", c, ids[0])
	}
	if got := courierStatus(t, c.ID); got != "busy" {
		t.Errorf("courier status in DB = %q, want busy", got)
	}

	d, err := testStore.DeliveryByOrder(ctx, 100)
	if err != nil {
		t.Fatalf("DeliveryByOrder: %v", err)
	}
	if d.UserID != 7 || d.CourierID != c.ID || d.Address != "Khreshchatyk 1" || d.Status != "assigned" || d.CompletedAt != nil {
		t.Errorf("delivery = %+v", d)
	}
	if time.Since(d.CreatedAt) > time.Minute {
		t.Errorf("created_at = %v, want about now", d.CreatedAt)
	}
}

func TestAssignCourierRedelivery(t *testing.T) {
	setup(t, "Ivan", "Olena")
	ctx := t.Context()

	first, _, err := testStore.AssignCourier(ctx, 100, 7, "Khreshchatyk 1")
	if err != nil {
		t.Fatal(err)
	}
	again, created, err := testStore.AssignCourier(ctx, 100, 7, "Khreshchatyk 1")
	if err != nil {
		t.Fatalf("second AssignCourier: %v", err)
	}
	if created {
		t.Error("created = true for the same order, want false")
	}
	if again.ID != first.ID {
		t.Errorf("got courier %d, want the originally assigned %d", again.ID, first.ID)
	}

	var n int
	if err := testStore.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deliveries`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("deliveries = %d, want 1", n)
	}
}

func TestAssignCourierNoneAvailable(t *testing.T) {
	setup(t, "Ivan")
	ctx := t.Context()

	if _, _, err := testStore.AssignCourier(ctx, 100, 7, "addr"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := testStore.AssignCourier(ctx, 101, 7, "addr"); !errors.Is(err, ErrNoCouriers) {
		t.Errorf("err = %v, want ErrNoCouriers", err)
	}
}

// Several workers in one consumer group assign couriers at the same time.
// SKIP LOCKED must give every order a different courier.
func TestAssignCourierConcurrent(t *testing.T) {
	const n = 8
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("courier-%d", i)
	}
	setup(t, names...)

	var (
		mu       sync.Mutex
		assigned = map[int64]int64{} // courier -> order
		wg       sync.WaitGroup
	)
	for order := int64(1); order <= n; order++ {
		wg.Go(func() {
			c, _, err := testStore.AssignCourier(t.Context(), order, 1, "addr")
			if err != nil {
				t.Errorf("order %d: %v", order, err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if prev, dup := assigned[c.ID]; dup {
				t.Errorf("courier %d assigned to orders %d and %d", c.ID, prev, order)
			}
			assigned[c.ID] = order
		})
	}
	wg.Wait()

	if len(assigned) != n {
		t.Errorf("%d distinct couriers assigned, want %d", len(assigned), n)
	}
}

func TestCompleteDelivery(t *testing.T) {
	setup(t, "Ivan")
	ctx := t.Context()

	c, _, err := testStore.AssignCourier(ctx, 100, 7, "addr")
	if err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 10, 27, 12, 30, 0, 0, time.UTC)
	d, err := testStore.CompleteDelivery(ctx, 100, at)
	if err != nil {
		t.Fatalf("CompleteDelivery: %v", err)
	}
	if d.Status != "completed" || d.UserID != 7 {
		t.Errorf("returned delivery = %+v", d)
	}
	if got := courierStatus(t, c.ID); got != "available" {
		t.Errorf("courier status = %q, want available", got)
	}

	stored, err := testStore.DeliveryByOrder(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "completed" || stored.CompletedAt == nil || !stored.CompletedAt.Equal(at) {
		t.Errorf("stored delivery = status %q, completed_at %v; want completed at %v", stored.Status, stored.CompletedAt, at)
	}

	if _, err := testStore.CompleteDelivery(ctx, 100, at.Add(time.Hour)); !errors.Is(err, ErrAlreadyDone) {
		t.Errorf("second CompleteDelivery err = %v, want ErrAlreadyDone", err)
	}
}

func TestNotFound(t *testing.T) {
	setup(t)
	ctx := t.Context()

	if _, err := testStore.DeliveryByOrder(ctx, 404); !errors.Is(err, ErrNotFound) {
		t.Errorf("DeliveryByOrder err = %v, want ErrNotFound", err)
	}
	if _, err := testStore.CompleteDelivery(ctx, 404, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Errorf("CompleteDelivery err = %v, want ErrNotFound", err)
	}
}

func TestCouriers(t *testing.T) {
	ctx := t.Context()

	setup(t)
	got, err := testStore.Couriers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("empty table: got %#v, want empty non-nil slice (encodes as [])", got)
	}

	setup(t, "Ivan", "Olena")
	if _, _, err := testStore.AssignCourier(ctx, 100, 7, "addr"); err != nil {
		t.Fatal(err)
	}
	got, err = testStore.Couriers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Ivan" || got[0].Status != "busy" || got[1].Name != "Olena" || got[1].Status != "available" {
		t.Errorf("couriers = %+v", got)
	}
}
