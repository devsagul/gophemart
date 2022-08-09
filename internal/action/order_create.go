package action

import (
	"log"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
)

func OrderCreate(orderId string, user *core.User, orderStorage storage.OrderStorage) error {
	order, err := core.NewOrder(orderId, user, time.Now())
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	err = orderStorage.Create(order)
	if err != nil {
		return err
	}
	return nil
}
