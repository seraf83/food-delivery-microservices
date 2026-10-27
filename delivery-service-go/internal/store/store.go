// Package store is the MySQL persistence layer for couriers and deliveries.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrNoCouriers  = errors.New("no available couriers")
	ErrAlreadyDone = errors.New("delivery already completed")
)

type Courier struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Phone  string `json:"-"`
	Status string `json:"status"`
}

type Delivery struct {
	ID          int64
	OrderID     int64
	UserID      int64
	CourierID   int64
	Address     string
	Status      string
	CompletedAt *time.Time
	CreatedAt   time.Time
}

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// MySQL may still be starting when the container comes up.
	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return &Store{db: db}, nil
		}
		if attempt == 20 {
			db.Close()
			return nil, fmt.Errorf("connect to mysql: %w", err)
		}
		select {
		case <-ctx.Done():
			db.Close()
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *Store) Close() error { return s.db.Close() }

// Migrate creates the tables if they do not exist. The schema matches the one
// the Doctrine migration created, so an existing database is reused as is.
func (s *Store) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS couriers (
			id INT AUTO_INCREMENT NOT NULL,
			name VARCHAR(100) NOT NULL,
			phone VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL,
			PRIMARY KEY (id)
		) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS deliveries (
			id INT AUTO_INCREMENT NOT NULL,
			order_id INT NOT NULL,
			user_id INT NOT NULL,
			courier_id INT NOT NULL,
			address VARCHAR(500) NOT NULL,
			status VARCHAR(50) NOT NULL,
			completed_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id)
		) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,
	}
	for _, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// AssignCourier atomically picks an available courier, marks them busy and
// creates the delivery. If a delivery for the order already exists (the event
// was redelivered) it returns that delivery's courier and created=false.
func (s *Store) AssignCourier(ctx context.Context, orderID, userID int64, address string) (c Courier, created bool, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Courier{}, false, err
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit

	var existingCourier int64
	err = tx.QueryRowContext(ctx,
		`SELECT courier_id FROM deliveries WHERE order_id = ? LIMIT 1`, orderID,
	).Scan(&existingCourier)
	switch {
	case err == nil:
		err = tx.QueryRowContext(ctx,
			`SELECT id, name, phone, status FROM couriers WHERE id = ?`, existingCourier,
		).Scan(&c.ID, &c.Name, &c.Phone, &c.Status)
		return c, false, err
	case !errors.Is(err, sql.ErrNoRows):
		return Courier{}, false, err
	}

	// SKIP LOCKED lets concurrent workers grab different couriers instead of
	// both picking the same one.
	err = tx.QueryRowContext(ctx,
		`SELECT id, name, phone, status FROM couriers
		 WHERE status = 'available' ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Courier{}, false, ErrNoCouriers
	}
	if err != nil {
		return Courier{}, false, err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE couriers SET status = 'busy' WHERE id = ?`, c.ID); err != nil {
		return Courier{}, false, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO deliveries (order_id, user_id, courier_id, address, status, created_at)
		 VALUES (?, ?, ?, ?, 'assigned', ?)`,
		orderID, userID, c.ID, address, time.Now().UTC(),
	)
	if err != nil {
		return Courier{}, false, err
	}

	c.Status = "busy"
	return c, true, tx.Commit()
}

// CompleteDelivery marks the order's delivery completed and frees the courier.
func (s *Store) CompleteDelivery(ctx context.Context, orderID int64, at time.Time) (Delivery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Delivery{}, err
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit

	d, err := scanDelivery(tx.QueryRowContext(ctx, selectDelivery+` WHERE order_id = ? LIMIT 1 FOR UPDATE`, orderID))
	if err != nil {
		return Delivery{}, err
	}
	if d.Status == "completed" {
		return d, ErrAlreadyDone
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE deliveries SET status = 'completed', completed_at = ? WHERE id = ?`, at, d.ID,
	); err != nil {
		return Delivery{}, err
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE couriers SET status = 'available' WHERE id = ?`, d.CourierID,
	); err != nil {
		return Delivery{}, err
	}

	d.Status = "completed"
	d.CompletedAt = &at
	return d, tx.Commit()
}

func (s *Store) DeliveryByOrder(ctx context.Context, orderID int64) (Delivery, error) {
	return scanDelivery(s.db.QueryRowContext(ctx, selectDelivery+` WHERE order_id = ? LIMIT 1`, orderID))
}

func (s *Store) Couriers(ctx context.Context) ([]Courier, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, phone, status FROM couriers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	couriers := []Courier{}
	for rows.Next() {
		var c Courier
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Status); err != nil {
			return nil, err
		}
		couriers = append(couriers, c)
	}
	return couriers, rows.Err()
}

const selectDelivery = `SELECT id, order_id, user_id, courier_id, address, status, completed_at, created_at FROM deliveries`

func scanDelivery(row *sql.Row) (Delivery, error) {
	var d Delivery
	var completedAt sql.NullTime
	err := row.Scan(&d.ID, &d.OrderID, &d.UserID, &d.CourierID, &d.Address, &d.Status, &completedAt, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Delivery{}, ErrNotFound
	}
	if err != nil {
		return Delivery{}, err
	}
	if completedAt.Valid {
		d.CompletedAt = &completedAt.Time
	}
	return d, nil
}
