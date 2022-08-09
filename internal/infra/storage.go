package infra

import "github.com/devsagul/gophemart/internal/storage"

type repository struct {
	users  storage.UserStorage
	orders storage.OrderStorage
}

func NewInMemoryRepository() repository {
	var res repository

	res.users = storage.NewUserMemStorage()
	res.orders = storage.NewOrderMemStorage()
	return res
}
