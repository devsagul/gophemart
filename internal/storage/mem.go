package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type memStorage struct {
	sync.RWMutex
	keys        map[uuid.UUID]core.HmacKey
	orders      map[string]core.Order
	users       map[string]core.User
	withdrawals map[uuid.UUID]core.Withdrawal
}

// Persist new hmac key within the storage
func (store *memStorage) CreateKey(key *core.HmacKey) error {
	store.Lock()
	defer store.Unlock()

	store.keys[key.ID] = *key
	return nil
}

// extract a key by id
func (store *memStorage) ExtractKey(id uuid.UUID) (*core.HmacKey, error) {
	store.RLock()
	defer store.RUnlock()

	key, found := store.keys[id]
	if !found || key.Expired() {
		return nil, &ErrKeyNotFound{id}
	}
	return &key, nil
}

// extract random valid fresh key (if any)
func (store *memStorage) ExtractRandomKey() (*core.HmacKey, error) {
	store.RLock()
	defer store.RUnlock()

	keys := []*core.HmacKey{}

	for _, key := range store.keys {
		key := key
		if key.Fresh() {
			keys = append(keys, &key)
		}
	}

	if len(keys) == 0 {
		return nil, &ErrNoKeys{}
	}

	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)
	i := r.Intn(len(keys))

	return keys[i], nil
}

// extract all keys
func (store *memStorage) ExtractAllKeys() (map[uuid.UUID]core.HmacKey, error) {
	store.RLock()
	defer store.RUnlock()

	keys := make(map[uuid.UUID]core.HmacKey)

	for _, key := range store.keys {
		keys[key.ID] = key
	}

	return keys, nil
}

// Persist new order item
func (store *memStorage) CreateOrder(order *core.Order) error {
	userID := order.UserID
	orderID := order.ID

	store.Lock()
	defer store.Unlock()

	prev, found := store.orders[orderID]
	if found {
		if prev.UserID == userID {
			return &ErrOrderExists{orderID}
		}
		return &ErrOrderIDCollission{orderID}
	}
	store.orders[orderID] = *order
	return nil
}

// Extract all orders by user
func (store *memStorage) ExtractOrdersByUser(user *core.User) ([]*core.Order, error) {
	userID := user.ID
	res := []*core.Order{}

	store.RLock()
	defer store.RUnlock()
	for _, order := range store.orders {
		order := order
		if order.UserID == userID {
			res = append(res, &order)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].UploadedAt.Before(res[j].UploadedAt)
	})

	return res, nil
}

// Extract all unterminated orders for all users
func (store *memStorage) ExtractUnterminatedOrders() ([]*core.Order, error) {
	orders := []*core.Order{}
	store.RLock()
	defer store.RUnlock()
	for _, order := range store.orders {
		order := order
		if order.Status != core.PROCESSED && order.Status != core.INVALID {
			orders = append(orders, &order)
		}
	}

	return orders, nil
}

// Persist new user
func (store *memStorage) CreateUser(user *core.User) error {
	login := user.Login

	store.Lock()
	defer store.Unlock()

	_, found := store.users[login]
	if found {
		return &ErrConflictingUserLogin{login}
	}
	store.users[login] = *user
	return nil
}

// Extract user by login
func (store *memStorage) ExtractUser(login string) (*core.User, error) {
	store.RLock()
	defer store.RUnlock()

	user, found := store.users[login]
	if !found {
		return nil, &ErrUserNotFound{login}
	}

	return &user, nil
}

// Extract user by id
func (store *memStorage) ExtractUserByID(id uuid.UUID) (*core.User, error) {
	store.RLock()
	defer store.RUnlock()

	var user *core.User = nil
	for _, u := range store.users {
		u := u
		if u.ID == id {
			user = &u
			break
		}
	}

	if user == nil {
		return nil, &ErrUserNotFoundByID{id}
	}

	return user, nil
}

// Persist new withdrawal item
func (store *memStorage) CreateWithdrawal(withdrawal *core.Withdrawal, order *core.Order) error {
	store.Lock()
	defer store.Unlock()
	orderID := order.ID

	userID := order.UserID

	var user *core.User = nil
	for _, u := range store.users {
		if u.ID == userID {
			user = &u
			break
		}
	}
	if user == nil {
		return &ErrUserNotFoundByID{userID}
	}

	if user.Balance.LessThan(withdrawal.Sum) {
		log.Printf("err: %v < %v; %v", user.Balance, withdrawal.Sum, user)
		return &ErrBalanceExceeded{}
	}

	prev, found := store.orders[orderID]
	if found {
		if prev.UserID == userID {
			return &ErrOrderExists{orderID}
		}
		return &ErrOrderIDCollission{orderID}
	}
	store.orders[orderID] = *order

	user.Balance = user.Balance.Sub(withdrawal.Sum)

	store.withdrawals[withdrawal.ID] = *withdrawal
	store.users[user.Login] = *user
	return nil
}

// Extract withdrawals by user
func (store *memStorage) ExtractWithdrawalsByUser(user *core.User) ([]*core.Withdrawal, error) {
	userOrders := make(map[string]bool)
	res := []*core.Withdrawal{}

	store.RLock()
	defer store.RUnlock()

	for _, order := range store.orders {
		if order.UserID == user.ID {
			userOrders[order.ID] = true
		}
	}

	for _, withdrawal := range store.withdrawals {
		withdrawal := withdrawal
		orderID := withdrawal.OrderID
		_, found := userOrders[orderID]
		if found {
			res = append(res, &withdrawal)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ProcessedAt.Before(res[j].ProcessedAt)
	})

	return res, nil
}

// Calculate total withdrawn sum for user
func (store *memStorage) TotalWithdrawnSum(user *core.User) (decimal.Decimal, error) {
	store.RLock()
	defer store.RUnlock()

	withdrawn := decimal.Zero
	for _, withdrawal := range store.withdrawals {
		orderID := withdrawal.OrderID
		order, found := store.orders[orderID]
		if !found {
			return decimal.Zero, errors.New("no order found")
		}
		if order.UserID == user.ID {
			withdrawn = withdrawn.Add(withdrawal.Sum)
		}
	}

	return withdrawn, nil
}

// Register new accrual within the storage
func (store *memStorage) ProcessAccrual(orderID string, status string, sum *decimal.Decimal) error {
	if status == "REGISTERED" {
		status = core.NEW
	}

	if status != core.NEW && status != core.PROCESSING && status != core.INVALID && status != core.PROCESSED {
		return fmt.Errorf("invalid order status: %v", status)
	}

	store.Lock()
	defer store.Unlock()

	order, found := store.orders[orderID]
	if !found {
		return fmt.Errorf("order with id %s does not exist", orderID)
	}

	order.Status = status

	id := order.UserID

	var user *core.User = nil
	for _, u := range store.users {
		u := u
		if u.ID == id {
			user = &u
			break
		}
	}

	if user == nil {
		return &ErrUserNotFoundByID{id}
	}

	if sum != nil {
		user.Balance = user.Balance.Add(*sum)
		order.Accrual = sum
	}

	store.orders[orderID] = order
	store.users[user.Login] = *user
	return nil
}

// Checks if storage is awailable (trivial for in-memory storage)
func (store *memStorage) Ping(context.Context) error {
	return nil
}

// Constructs new storage object with given context
func (store *memStorage) WithContext(context.Context) Storage {
	return store
}

// Create new in-memory storage
func NewMemStorage() *memStorage {
	store := new(memStorage)
	store.keys = make(map[uuid.UUID]core.HmacKey)
	store.orders = make(map[string]core.Order)
	store.users = make(map[string]core.User)
	store.withdrawals = make(map[uuid.UUID]core.Withdrawal)
	return store
}
