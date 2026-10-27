// Package worker consumes OrderPlaced events and assigns couriers.
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/seraf83/food-delivery/delivery-service/internal/config"
	"github.com/seraf83/food-delivery/delivery-service/internal/events"
	"github.com/seraf83/food-delivery/delivery-service/internal/store"
)

type Store interface {
	AssignCourier(ctx context.Context, orderID, userID int64, address string) (store.Courier, bool, error)
}

type Publisher interface {
	Publish(ctx context.Context, typ string, data any) error
}

// Reader is the subset of *kafka.Reader the worker uses.
type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type Worker struct {
	reader Reader
	store  Store
	pub    Publisher
	log    *slog.Logger
}

func New(cfg config.Config, st Store, pub Publisher, log *slog.Logger) *Worker {
	log.Info("consuming", "topic", cfg.OrdersTopic, "group", cfg.ConsumerGroup)
	return &Worker{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     cfg.KafkaBrokers,
			Topic:       cfg.OrdersTopic,
			GroupID:     cfg.ConsumerGroup,
			StartOffset: kafka.FirstOffset,
		}),
		store: st,
		pub:   pub,
		log:   log,
	}
}

// Run processes messages until ctx is cancelled. An offset is committed only
// after the message is handled, so a crash mid-way means a redelivery, which
// AssignCourier tolerates.
func (w *Worker) Run(ctx context.Context) error {
	defer w.reader.Close()

	for {
		msg, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		if err := w.handle(ctx, msg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// Transient failure (DB/Kafka): don't commit; exit so the container
			// restarts and re-reads the message.
			w.log.Error("handle message", "offset", msg.Offset, "err", err)
			return err
		}

		if err := w.reader.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
			return err
		}
	}
}

func (w *Worker) handle(ctx context.Context, msg kafka.Message) error {
	var env events.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil || env.Type == "" {
		w.log.Warn("invalid message format, skipping", "offset", msg.Offset)
		return nil
	}
	if env.Type != events.TypeOrderPlaced {
		w.log.Debug("ignoring message", "type", env.Type)
		return nil
	}

	var ev events.OrderPlaced
	if err := json.Unmarshal(env.Data, &ev); err != nil {
		w.log.Warn("invalid OrderPlaced payload, skipping", "offset", msg.Offset, "err", err)
		return nil
	}

	w.log.Info("📦 new order received", "order", ev.OrderID)

	courier, created, err := w.store.AssignCourier(ctx, ev.OrderID, ev.UserID, ev.DeliveryAddress)
	if errors.Is(err, store.ErrNoCouriers) {
		w.log.Warn("⚠️  no available couriers", "order", ev.OrderID)
		return nil
	}
	if err != nil {
		return err
	}
	if !created {
		// Duplicate OrderPlaced: re-publishing would let a stale DeliveryAssigned
		// overwrite a later status (e.g. delivered) in order-service.
		w.log.Info("delivery already exists, skipping duplicate", "order", ev.OrderID)
		return nil
	}

	publishCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	err = w.pub.Publish(publishCtx, events.TypeDeliveryAssigned, events.DeliveryAssigned{
		OrderID:          ev.OrderID,
		CourierID:        courier.ID,
		CourierName:      courier.Name,
		CourierPhone:     courier.Phone,
		EstimatedMinutes: rand.IntN(3) + 1,
		UserID:           ev.UserID,
	})
	if err != nil {
		return err
	}

	w.log.Info("🛵 courier assigned", "courier", courier.Name, "order", ev.OrderID)
	return nil
}
