package action

import (
	"log"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
)

func OrderCreate(orderId string, user *core.User, store storage.Storage) error {
	order, err := core.NewOrder(orderId, user, time.Now())
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	err = store.CreateOrder(order)
	if err != nil {
		return err
	}
	return nil
}
