package core

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Withdrawal struct {
	Id          uuid.UUID       `json:"-"`
	OrderId     string          `json:"order"`
	Sum         decimal.Decimal `json:"sum"`
	ProcessedAt time.Time       `json:"processed_at"`
}

func NewWithdrawal(order *Order, sum decimal.Decimal, processedAt time.Time) (*Withdrawal, error) {
	// TODO check if order is nil
	withdrawal := new(Withdrawal)
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	withdrawal.Id = id
	withdrawal.OrderId = order.Id
	withdrawal.Sum = sum
	withdrawal.ProcessedAt = processedAt
	return withdrawal, nil
}
