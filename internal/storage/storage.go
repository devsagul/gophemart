// Storage package provides different implementation of storages for the entities in the application

package storage

import (
	"context"
	"fmt"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Storage for secret keys (auth-related)
type AuthStorage interface {
	// Persist new hmac key within the storage
	CreateKey(*core.HmacKey) error
	// extract a key by id
	ExtractKey(uuid.UUID) (*core.HmacKey, error)
	// extract random valid fresh key (if any)
	ExtractRandomKey() (*core.HmacKey, error)
	// extract all keys
	ExtractAllKeys() (map[uuid.UUID]core.HmacKey, error)
}

// Storage for orders
type OrdersStorage interface {
	// Persist new order item
	CreateOrder(*core.Order) error
	// Extract all orders by user
	ExtractOrdersByUser(*core.User) ([]*core.Order, error)
	// Extract all unterminated orders for all users
	ExtractUnterminatedOrders() ([]*core.Order, error)
}

// Storage for users
type UsersStorage interface {
	// Persist new user
	CreateUser(*core.User) error
	// Extract user by login
	ExtractUser(string) (*core.User, error)
	// Extract user by id
	ExtractUserByID(uuid.UUID) (*core.User, error)
}

// Wuthdrawal storage
type WithdrawalsStorage interface {
	// Persist new withdrawal item
	CreateWithdrawal(*core.Withdrawal, *core.Order) error
	// Extract withdrawals by user
	ExtractWithdrawalsByUser(*core.User) ([]*core.Withdrawal, error)
	// Calculate total withdrawn sum for user
	TotalWithdrawnSum(*core.User) (decimal.Decimal, error)
}

// Storage for accruals
type AccrualStorage interface {
	// Register new accrual within a storage
	ProcessAccrual(orderID string, status string, sum *decimal.Decimal) error
}

// General storage implementation interface
type Storage interface {
	// Checks if storage is awailable
	Ping(context.Context) error
	// Constructs new storage object with given context
	WithContext(context.Context) Storage
	// auth
	AuthStorage
	// orders
	OrdersStorage
	// users
	UsersStorage
	// withdrawals
	WithdrawalsStorage
	// accrual
	AccrualStorage
}

// errors

// auth

// Error: HMAC key is not found by id
type ErrKeyNotFound struct {
	keyID uuid.UUID
}

// Formats ErrKeyNotFound
func (err *ErrKeyNotFound) Error() string {
	return fmt.Sprintf("active key with id %s not found", err.keyID)
}

// Error: there are no fresh keys in the storage
type ErrNoKeys struct{}

// Formats ErrNoKeys
func (err *ErrNoKeys) Error() string {
	return "there are no active keys in storage"
}

// order

// Error: order with given ID exists for current user
type ErrOrderExists struct {
	orderID string
}

// Formats ErrOrderExists
func (err *ErrOrderExists) Error() string {
	return fmt.Sprintf("order with id %s exists already for current user", err.orderID)
}

// Error: order with given ID exists for another user
type ErrOrderIDCollission struct {
	orderID string
}

// Formats ErrOrderIDCollission
func (err *ErrOrderIDCollission) Error() string {
	return fmt.Sprintf("order with id %s exists already for other user", err.orderID)
}

// user

// Error: user can't be found by login
type ErrUserNotFound struct {
	login string
}

// Formats ErrUserNotFound
func (err *ErrUserNotFound) Error() string {
	return fmt.Sprintf("could not find user with login %s", err.login)
}

// Error: user can't be found by ID
type ErrUserNotFoundByID struct {
	id uuid.UUID
}

// Formats ErrUserNotFoundByID
func (err *ErrUserNotFoundByID) Error() string {
	return fmt.Sprintf("could not find user with id %s", err.id)
}

// Error: user with given login is present in the storage already
type ErrConflictingUserLogin struct {
	login string
}

// Formats ErrConflictingUserLogin
func (err *ErrConflictingUserLogin) Error() string {
	return fmt.Sprintf("conflicting user login %s", err.login)
}

// withdrawals

// Error: cannot persist withdrawal, because the amount requested exceeds user's balance
type ErrBalanceExceeded struct{}

// Formats ErrBalanceExceeded
func (err *ErrBalanceExceeded) Error() string {
	return "requested withdrawal amount exceeds user's balance"
}
