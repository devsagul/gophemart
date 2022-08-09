package storage

import (
	"sync"

	"github.com/devsagul/gophemart/internal/core"
)

type orderMemStorage struct {
	sync.Mutex
	m map[string]core.Order
}

func (store *orderMemStorage) Create(order *core.Order) error {
	userId := order.UserId
	orderId := order.Id

	store.Lock()
	defer store.Unlock()

	prev, found := store.m[orderId]
	if found {
		if prev.UserId == userId {
			return ErrOrderExitst
		}
		return ErrOrderIdCollision
	}
	store.m[orderId] = *order
	return nil
}

func NewOrderMemStorage() *orderMemStorage {
	storage := new(orderMemStorage)
	storage.m = make(map[string]core.Order)
	return storage
}
