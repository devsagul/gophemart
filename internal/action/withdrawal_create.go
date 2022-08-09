package action

import (
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/shopspring/decimal"
)

func WithdrawalCreate(user *core.User, orderId string, sum decimal.Decimal, store storage.Storage) error {
	_, err := core.NewOrder(orderId, user, time.Now())
	if err != nil {
		return err
	}
	return nil
}
