// Package events defines the Kafka message contract shared with the PHP services.
//
// Messages are JSON envelopes {"type": "<PHP class>", "data": {...}}. The PHP
// consumers build the message with `new $class(...array_values($data))`, so the
// field order in each struct below must match the PHP constructor exactly.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TypeOrderPlaced       = `App\Message\OrderPlaced`
	TypeDeliveryAssigned  = `App\Message\DeliveryAssigned`
	TypeDeliveryCompleted = `App\Message\DeliveryCompleted`
)

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type OrderPlaced struct {
	OrderID         int64           `json:"orderId"`
	UserID          int64           `json:"userId"`
	DeliveryAddress string          `json:"deliveryAddress"`
	Items           json.RawMessage `json:"items"`
	TotalAmount     float64         `json:"totalAmount"`
}

type DeliveryAssigned struct {
	OrderID          int64  `json:"orderId"`
	CourierID        int64  `json:"courierId"`
	CourierName      string `json:"courierName"`
	CourierPhone     string `json:"courierPhone"`
	EstimatedMinutes int    `json:"estimatedMinutes"`
	UserID           int64  `json:"userId"` // appended last: PHP maps fields by position
}

type DeliveryCompleted struct {
	OrderID     int64  `json:"orderId"`
	UserID      int64  `json:"userId"`
	CompletedAt string `json:"completedAt"` // "2006-01-02 15:04:05"
}

type Publisher struct {
	w *kafka.Writer
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{w: &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		WriteTimeout:           10 * time.Second,
	}}
}

func (p *Publisher) Publish(ctx context.Context, typ string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body, err := json.Marshal(Envelope{Type: typ, Data: payload})
	if err != nil {
		return err
	}

	err = p.w.WriteMessages(ctx, kafka.Message{
		Value:   body,
		Headers: []kafka.Header{{Key: "Content-Type", Value: []byte("application/json")}},
	})
	if err != nil {
		return fmt.Errorf("publish %s: %w", typ, err)
	}
	return nil
}

func (p *Publisher) Close() error { return p.w.Close() }
