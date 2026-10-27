package events

import (
	"encoding/json"
	"testing"
)

// PHP consumers map `data` onto constructor arguments by position, so the JSON
// key order is part of the contract.
func TestPayloadFieldOrder(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want string
	}{
		{
			"DeliveryAssigned",
			DeliveryAssigned{OrderID: 1, CourierID: 2, CourierName: "Ivan", CourierPhone: "+380", EstimatedMinutes: 3, UserID: 7},
			`{"orderId":1,"courierId":2,"courierName":"Ivan","courierPhone":"+380","estimatedMinutes":3,"userId":7}`,
		},
		{
			"DeliveryCompleted",
			DeliveryCompleted{OrderID: 1, UserID: 7, CompletedAt: "2026-10-15 12:00:00"},
			`{"orderId":1,"userId":7,"completedAt":"2026-10-15 12:00:00"}`,
		},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.v)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

func TestDecodePHPOrderPlaced(t *testing.T) {
	raw := `{"type":"App\\Message\\OrderPlaced","data":{"orderId":5,"userId":9,"deliveryAddress":"Khreshchatyk 1","items":[{"name":"Pizza","qty":1}],"totalAmount":250.5}}`

	var env Envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatal(err)
	}
	if env.Type != TypeOrderPlaced {
		t.Fatalf("type = %q", env.Type)
	}
	var ev OrderPlaced
	if err := json.Unmarshal(env.Data, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.OrderID != 5 || ev.UserID != 9 || ev.DeliveryAddress != "Khreshchatyk 1" || ev.TotalAmount != 250.5 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}
