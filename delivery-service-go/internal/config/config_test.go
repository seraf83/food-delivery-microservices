package config

import "testing"

func TestMySQLDSN(t *testing.T) {
	const opts = "?parseTime=true&loc=UTC&charset=utf8mb4"

	cases := []struct {
		name, in, want string
		wantErr        bool
	}{
		{"full", "mysql://delivery_user:delivery_pass@delivery-db:3306/delivery", "delivery_user:delivery_pass@tcp(delivery-db:3306)/delivery" + opts, false},
		{"default port", "mysql://u:p@db/delivery", "u:p@tcp(db:3306)/delivery" + opts, false},
		{"escaped password", "mysql://u:p%40ss@db:3306/delivery", "u:p@ss@tcp(db:3306)/delivery" + opts, false},
		{"empty", "", "", true},
		{"wrong scheme", "postgres://u:p@db:5432/delivery", "", true},
		{"not a URL", "://", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mysqlDSN(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://u:p@db:3306/delivery")
	for _, k := range []string{"HTTP_ADDR", "KAFKA_DSN", "KAFKA_ORDERS_TOPIC", "KAFKA_DELIVERIES_TOPIC", "KAFKA_GROUP_ID"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":80" || cfg.OrdersTopic != "orders" || cfg.DeliveriesTopic != "deliveries" || cfg.ConsumerGroup != "delivery-consumer" {
		t.Errorf("defaults = %+v", cfg)
	}
	if len(cfg.KafkaBrokers) != 1 || cfg.KafkaBrokers[0] != "kafka:9092" {
		t.Errorf("KafkaBrokers = %v", cfg.KafkaBrokers)
	}
}

func TestLoadBrokerList(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://u:p@db:3306/delivery")
	t.Setenv("KAFKA_DSN", "k1:9092,k2:9092")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[1] != "k2:9092" {
		t.Errorf("KafkaBrokers = %v, want [k1:9092 k2:9092]", cfg.KafkaBrokers)
	}
}
