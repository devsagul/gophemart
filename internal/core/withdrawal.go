package core

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Withdrawal struct {
	ID          uuid.UUID       `json:"-"`
	OrderID     string          `json:"order"`
	Sum         decimal.Decimal `json:"sum"`
	ProcessedAt time.Time       `json:"processed_at"`
}

func NewWithdrawal(order *Order, sum decimal.Decimal, processedAt time.Time) (*Withdrawal, error) {
	withdrawal := new(Withdrawal)
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	withdrawal.ID = id
	withdrawal.OrderID = order.ID
	withdrawal.Sum = sum
	withdrawal.ProcessedAt = processedAt
	return withdrawal, nil
}
