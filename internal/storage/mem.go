package storage

import (
	"errors"
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
	// todo extract types
	keys        map[uuid.UUID]core.HmacKey
	orders      map[string]core.Order
	users       map[string]core.User
	withdrawals map[uuid.UUID]core.Withdrawal
}

func (store *memStorage) CreateKey(key *core.HmacKey) error {
	store.Lock()
	defer store.Unlock()

	store.keys[key.Id] = *key
	return nil
}

func (store *memStorage) ExtractKey(id uuid.UUID) (*core.HmacKey, error) {
	store.RLock()
	defer store.RUnlock()

	key, found := store.keys[id]
	if !found || key.Expired() {
		return nil, &ErrKeyNotFound{id}
	}
	return &key, nil
}

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

func (store *memStorage) ExtractAllKeys() (map[uuid.UUID]core.HmacKey, error) {
	store.RLock()
	defer store.RUnlock()

	keys := make(map[uuid.UUID]core.HmacKey)

	for _, key := range store.keys {
		keys[key.Id] = key
	}

	return keys, nil
}

func (store *memStorage) CreateOrder(order *core.Order) error {
	userId := order.UserId
	orderId := order.Id

	store.Lock()
	defer store.Unlock()

	prev, found := store.orders[orderId]
	if found {
		if prev.UserId == userId {
			return &ErrOrderExists{orderId}
		}
		return &ErrOrderIdCollission{orderId}
	}
	store.orders[orderId] = *order
	return nil
}

func (store *memStorage) ExtractOrdersByUser(user *core.User) ([]*core.Order, error) {
	// TODO Add test
	userId := user.Id
	res := []*core.Order{}

	store.RLock()
	defer store.RUnlock()
	for _, order := range store.orders {
		order := order
		if order.UserId == userId {
			res = append(res, &order)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].UploadedAt.Before(res[j].UploadedAt)
	})

	return res, nil
}

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

func (store *memStorage) ExtractUser(login string) (*core.User, error) {
	store.RLock()
	defer store.RUnlock()

	user, found := store.users[login]
	if !found {
		return nil, &ErrUserNotFound{login}
	}

	return &user, nil
}

func (store *memStorage) ExtractUserById(id uuid.UUID) (*core.User, error) {
	store.RLock()
	defer store.RUnlock()

	var user *core.User = nil
	for _, u := range store.users {
		u := u
		if u.Id == id {
			user = &u
			break
		}
	}

	if user == nil {
		return nil, &ErrUserNotFoundById{id}
	}

	return user, nil
}

func (store *memStorage) PersistUser(user *core.User) error {
	store.Lock()
	defer store.Unlock()

	store.users[user.Login] = *user
	return nil
}

func (store *memStorage) CreateWithdrawal(withdrawal *core.Withdrawal) error {
	store.Lock()
	defer store.Unlock()
	orderId := withdrawal.OrderId

	order, found := store.orders[orderId]
	if !found {
		// todo new error type
		return errors.New("generic order error")
	}

	userId := order.UserId

	var user *core.User = nil
	for _, u := range store.users {
		if u.Id == userId {
			user = &u
			break
		}
	}
	if user == nil {
		// todo new error type
		return errors.New("generic user error")
	}

	if user.Balance.LessThan(withdrawal.Sum) {
		return &ErrBalanceExceeded{}
	}
	user.Balance = user.Balance.Sub(withdrawal.Sum)

	store.withdrawals[withdrawal.Id] = *withdrawal
	store.users[user.Login] = *user
	return nil
}

func (store *memStorage) ExtractWithdrawalsByUser(user *core.User) ([]*core.Withdrawal, error) {
	userOrders := make(map[string]bool)
	res := []*core.Withdrawal{}

	store.RLock()
	defer store.RUnlock()

	for _, order := range store.orders {
		if order.UserId == user.Id {
			userOrders[order.Id] = true
		}
	}

	for _, withdrawal := range store.withdrawals {
		withdrawal := withdrawal
		orderId := withdrawal.OrderId
		_, found := userOrders[orderId]
		if found {
			res = append(res, &withdrawal)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ProcessedAt.Before(res[j].ProcessedAt)
	})

	return res, nil
}

func (store *memStorage) TotalWithdrawnSum(user *core.User) (decimal.Decimal, error) {
	store.RLock()
	defer store.RUnlock()

	withdrawn := decimal.Zero
	for _, withdrawal := range store.withdrawals {
		orderId := withdrawal.OrderId
		order, found := store.orders[orderId]
		if !found {
			return decimal.Zero, errors.New("no order found")
		}
		if order.UserId == user.Id {
			withdrawn = withdrawn.Add(withdrawal.Sum)
		}
	}

	return withdrawn, nil
}

func NewMemStorage() Storage {
	store := new(memStorage)
	store.keys = make(map[uuid.UUID]core.HmacKey)
	store.orders = make(map[string]core.Order)
	store.users = make(map[string]core.User)
	store.withdrawals = make(map[uuid.UUID]core.Withdrawal)
	return store
}
