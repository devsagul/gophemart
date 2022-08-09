package storage

import (
	"errors"

	"github.com/devsagul/gophemart/internal/core"
)

var ErrOrderExitst = errors.New("order with given id exists already for current user")
var ErrOrderIdCollision = errors.New("order with given id exists already for other user")

type OrderStorage interface {
	Create(*core.Order) error
	ExtractByUser(*core.User) ([]*core.Order, error)
}
