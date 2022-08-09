package action

import (
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/shopspring/decimal"
)

func WithdrawalCreate(user *core.User, orderId string, sum decimal.Decimal, store storage.Storage) error {
	timestamp := time.Now()
	order, err := core.NewOrder(orderId, user, timestamp)
	if err != nil {
		return err
	}
	withdrawal, err := core.NewWithdrawal(order, sum, timestamp)
	if err != nil {
		return err
	}
	err = store.CreateWithdrawal(withdrawal, order)
	return err
}
