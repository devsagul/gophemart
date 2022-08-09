package storage

import (
	"sort"
	"sync"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/google/uuid"
)

type memStorage struct {
	sync.RWMutex
	orders      map[string]core.Order
	users       map[string]core.User
	withdrawals map[uuid.UUID]core.Withdrawal
}

func (store *memStorage) CreateOrder(order *core.Order) error {
	userId := order.UserId
	orderId := order.Id

	store.Lock()
	defer store.Unlock()

	prev, found := store.orders[orderId]
	if found {
		if prev.UserId == userId {
			return ErrOrderExitst
		}
		return ErrOrderIdCollision
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

func (store *memStorage) PersistUser(user *core.User) error {
	store.Lock()
	defer store.Unlock()

	store.users[user.Login] = *user
	return nil
}

func (store *memStorage) CreateWithdrawal(withdrawal *core.Withdrawal, order *core.Order) error {
	store.Lock()
	defer store.Unlock()
	userId := order.UserId
	orderId := order.Id

	prev, found := store.orders[orderId]
	if found {
		if prev.UserId == userId {
			return ErrOrderExitst
		}
		return ErrOrderIdCollision
	}

	var user *core.User = nil
	for _, u := range store.users {
		if u.Id == userId {
			user = &u
			break
		}
	}
	if user == nil {
		// could not connect user
	}
	if user.Balance.LessThan(withdrawal.Sum) {
		// not enough money
	}
	store.orders[order.Id] = *order
	store.withdrawals[withdrawal.Id] = *withdrawal
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
		orderId := withdrawal.OrderId
		_, found := userOrders[orderId]
		if found {
			res = append(res, &withdrawal)
		}
	}

	return res, nil
}

func NewMemStorage() Storage {
	store := new(memStorage)
	store.orders = make(map[string]core.Order)
	store.users = make(map[string]core.User)
	store.withdrawals = make(map[uuid.UUID]core.Withdrawal)
	return store
}
