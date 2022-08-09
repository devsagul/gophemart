package action

import (
	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
)

func OrderList(user *core.User, store storage.Storage) ([]*core.Order, error) {
	return store.ExtractOrdersByUser(user)
}
