package messages

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Heartbeat struct {
	CurrentTime time.Time
}

type OrderStatus string

const (
	Open      OrderStatus = "OPEN"
	Partial   OrderStatus = "PARTIAL"
	Filled    OrderStatus = "FILLED"
	Cancelled OrderStatus = "CANCELLED"
)

func ParseOrderStatus(status string) (OrderStatus, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")

	switch normalized {
	case "open":
		return Open, nil
	case "partial", "partially_filled", "partiallyfilled":
		return Partial, nil
	case "filled", "complete", "completed", "closed":
		return Filled, nil
	case "cancelled", "canceled", "cancelled_partial", "canceled_partial", "expired":
		return Cancelled, nil
	default:
		return "", errors.New("unknown order status")
	}
}

type Order struct {
	ID      string
	Status  OrderStatus
	Account string
}

type Transaction struct {
	ID      string
	Status  string
	Account string
}

type Trade struct {
	Market    string
	Side      string
	Price     decimal.Decimal
	ID        string
	Timestamp time.Time
	Quantity  decimal.Decimal
}

type Change struct {
	Market    string
	Side      string
	Price     decimal.Decimal
	Remaining decimal.Decimal
	Delta     decimal.Decimal
	Action    string
}
