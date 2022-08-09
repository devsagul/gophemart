package storage

import (
	"sort"
	"sync"

	"github.com/devsagul/gophemart/internal/core"
)

type orderMemStorage struct {
	sync.RWMutex
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

func (store *orderMemStorage) ExtractByUser(user *core.User) ([]*core.Order, error) {
	// TODO Add test
	userId := user.Id
	res := []*core.Order{}

	store.RLock()
	defer store.RUnlock()
	for _, order := range store.m {
		if order.UserId == userId {
			res = append(res, &order)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].UploadedAt.Before(res[j].UploadedAt)
	})

	return res, nil
}

func NewOrderMemStorage() *orderMemStorage {
	storage := new(orderMemStorage)
	storage.m = make(map[string]core.Order)
	return storage
}
