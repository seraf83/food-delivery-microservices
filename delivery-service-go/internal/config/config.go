// Package config reads service settings from environment variables.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr        string
	DatabaseDSN     string // go-sql-driver/mysql format
	KafkaBrokers    []string
	OrdersTopic     string
	DeliveriesTopic string
	ConsumerGroup   string
}

func Load() (Config, error) {
	dsn, err := mysqlDSN(os.Getenv("DATABASE_URL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:        env("HTTP_ADDR", ":80"),
		DatabaseDSN:     dsn,
		KafkaBrokers:    strings.Split(env("KAFKA_DSN", "kafka:9092"), ","),
		OrdersTopic:     env("KAFKA_ORDERS_TOPIC", "orders"),
		DeliveriesTopic: env("KAFKA_DELIVERIES_TOPIC", "deliveries"),
		ConsumerGroup:   env("KAFKA_GROUP_ID", "delivery-consumer"),
	}, nil
}

// mysqlDSN converts mysql://user:pass@host:port/db (the format the PHP services
// use) into user:pass@tcp(host:port)/db.
func mysqlDSN(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("DATABASE_URL is not set")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "mysql" {
		return "", fmt.Errorf("DATABASE_URL must look like mysql://user:pass@host:port/db")
	}

	pass, _ := u.User.Password()
	host := u.Host
	if u.Port() == "" {
		host += ":3306"
	}

	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4",
		u.User.Username(), pass, host, strings.TrimPrefix(u.Path, "/")), nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
