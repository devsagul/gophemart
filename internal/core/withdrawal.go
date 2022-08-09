package core

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Withdrawal struct {
	// todo add json tags
	Id          uuid.UUID
	OrderId     string
	Sum         decimal.Decimal
	ProcessedAt time.Time
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
