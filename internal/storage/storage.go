package storage

import (
	"errors"
	"fmt"

	"github.com/devsagul/gophemart/internal/core"
)

var ErrOrderExitst = errors.New("order with given id exists already for current user")
var ErrOrderIdCollision = errors.New("order with given id exists already for other user")

type Storage interface {
	// orders
	CreateOrder(*core.Order) error
	ExtractOrdersByUser(*core.User) ([]*core.Order, error)
	// users
	CreateUser(*core.User) error
	ExtractUser(login string) (*core.User, error)
}

// errors

// order
type ErrOrderExists struct {
	orderId string
}

func (err *ErrOrderExists) Error() string {
	return fmt.Sprintf("order with id %s exists already for current user", err.orderId)
}

type ErrOrderIdCollission struct {
	orderId string
}

func (err *ErrOrderIdCollission) Error() string {
	return fmt.Sprintf("order with id %s exists already for other user", err.orderId)
}

// user
type ErrUserNotFound struct {
	login string
}

func (err *ErrUserNotFound) Error() string {
	return fmt.Sprintf("could not find user with login %s", err.login)
}

type ErrConflictingUserLogin struct {
	login string
}

func (err *ErrConflictingUserLogin) Error() string {
	return fmt.Sprintf("conflicting user login %s", err.login)
}
