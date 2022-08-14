package storage

import (
	"context"
	"fmt"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Storage interface {
	Ping(context.Context) error
	WithContext(context.Context) Storage
	// auth
	CreateKey(*core.HmacKey) error
	ExtractKey(uuid.UUID) (*core.HmacKey, error)
	ExtractRandomKey() (*core.HmacKey, error)
	ExtractAllKeys() (map[uuid.UUID]core.HmacKey, error)
	// orders
	CreateOrder(*core.Order) error
	ExtractOrdersByUser(*core.User) ([]*core.Order, error)
	ExtractUnterminatedOrders() ([]*core.Order, error)
	// users
	CreateUser(*core.User) error
	ExtractUser(string) (*core.User, error)
	ExtractUserByID(uuid.UUID) (*core.User, error)
	// withdrawals
	CreateWithdrawal(*core.Withdrawal, *core.Order) error
	ExtractWithdrawalsByUser(*core.User) ([]*core.Withdrawal, error)
	TotalWithdrawnSum(*core.User) (decimal.Decimal, error)
	// accrual
	ProcessAccrual(orderID string, status string, sum *decimal.Decimal) error
}

// errors

// auth
type ErrKeyNotFound struct {
	keyID uuid.UUID
}

func (err *ErrKeyNotFound) Error() string {
	return fmt.Sprintf("active key with id %s not found", err.keyID)
}

type ErrNoKeys struct{}

func (err *ErrNoKeys) Error() string {
	return "there are no active keys in storage"
}

// order
type ErrOrderExists struct {
	orderID string
}

func (err *ErrOrderExists) Error() string {
	return fmt.Sprintf("order with id %s exists already for current user", err.orderID)
}

type ErrOrderIDCollission struct {
	orderID string
}

func (err *ErrOrderIDCollission) Error() string {
	return fmt.Sprintf("order with id %s exists already for other user", err.orderID)
}

// user
type ErrUserNotFound struct {
	login string
}

func (err *ErrUserNotFound) Error() string {
	return fmt.Sprintf("could not find user with login %s", err.login)
}

type ErrUserNotFoundByID struct {
	id uuid.UUID
}

func (err *ErrUserNotFoundByID) Error() string {
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
