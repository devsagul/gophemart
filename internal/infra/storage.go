package infra

import "github.com/devsagul/gophemart/internal/storage"

type repository struct {
	users storage.UserStorage
}

func NewInMemoryRepository() repository {
	var res repository

	res.users = storage.NewUserMemStorage()
	return res
}
