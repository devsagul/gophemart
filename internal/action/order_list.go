package action

import (
	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
)

func OrderList(user *core.User, orderStorage storage.OrderStorage) ([]*core.Order, error) {
	return orderStorage.ExtractByUser(user)
}
