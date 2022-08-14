package storage

import (
	"fmt"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Storage interface {
	// auth
	CreateKey(*core.HmacKey) error
	ExtractKey(uuid.UUID) (*core.HmacKey, error)
	ExtractRandomKey() (*core.HmacKey, error)
	ExtractAllKeys() (map[uuid.UUID]core.HmacKey, error)
	// orders
	CreateOrder(*core.Order) error
	ExtractOrdersByUser(*core.User) ([]*core.Order, error)
	// users
	CreateUser(*core.User) error
	ExtractUser(string) (*core.User, error)
	ExtractUserById(uuid.UUID) (*core.User, error)
	// withdrawals
	CreateWithdrawal(*core.Withdrawal, *core.Order) error
	ExtractWithdrawalsByUser(*core.User) ([]*core.Withdrawal, error)
	TotalWithdrawnSum(*core.User) (decimal.Decimal, error)
}

// errors

// auth
type ErrKeyNotFound struct {
	keyId uuid.UUID
}

func (err *ErrKeyNotFound) Error() string {
	return fmt.Sprintf("active key with id %s not found", err.keyId)
}

type ErrNoKeys struct{}

func (err *ErrNoKeys) Error() string {
	return "there are no active keys in storage"
}

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

type ErrUserNotFoundById struct {
	id uuid.UUID
}

func (err *ErrUserNotFoundById) Error() string {
	return fmt.Sprintf("could not find user with id %s", err.id)
}

type ErrConflictingUserLogin struct {
	login string
}

func (err *ErrConflictingUserLogin) Error() string {
	return fmt.Sprintf("conflicting user login %s", err.login)
}

// withdrawals
type ErrBalanceExceeded struct{}

func (err *ErrBalanceExceeded) Error() string {
	return "requested withdrawal amount exceeds user's balance"
}
